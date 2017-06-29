/*
 * Copyright (c) 2017 Couchbase, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"../../logger"
	pb "../../rpc"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/codahale/hdrhistogram"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type AgentsConfig struct {
	agent map[string]string
}

type Coordinator struct {
	config             *Config
	agentsInfo         map[string]*AgentInfo
	db                 *sql.DB
	insertStatementStr string
	histogram          *hdrhistogram.Histogram
	logger             *logger.Logger
}

type AgentInfo struct {
	index    int
	hostname string
	conn     *grpc.ClientConn
	client   pb.AgentServiceClient
	results  map[string]*pb.AgentResultsResponse_CaptureInfo
}

type LatencyInfo struct {
	nodeType string
	opaque   string
	latency  string
	key      string
}

func (c *Coordinator) homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if html, err := ioutil.ReadFile("./graphplotter/index.html"); err != nil {
		c.logger.Error("%v", err)
		c.shutdown()
	} else {
		buffer := bytes.NewBuffer(make([]byte, 0, 1024))
		jsonStr, err := c.getFullCaptureFromDb()
		if err != nil {
			c.logger.Error("Unable to full capture results from db due to %v", err)
			os.Exit(1)
		}

		var agents []string
		for agent, _ := range c.agentsInfo {
			agents = append(agents, agent)
		}

		agentsJson, err := json.Marshal(agents)
		if err != nil {
			c.logger.Error("%v", err)
			c.shutdown()
		}

		buffer.WriteString("<script type=\"text/javascript\">")
		buffer.WriteString("var data=")
		buffer.WriteString(jsonStr)
		buffer.WriteString(";")
		buffer.WriteString("var yMax=")
		buffer.WriteString(strconv.FormatInt(c.getMaxLatency(), 10))
		buffer.WriteString(";")
		buffer.WriteString("var agents=")
		buffer.WriteString(string(agentsJson))
		buffer.WriteString("</script>")
		buffer.Write(html)

		w.Write(buffer.Bytes())
	}
}

func (c *Coordinator) startRestServer() {
	r := mux.NewRouter()
	r.HandleFunc("/", c.homeHandler)
	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprint(":", c.config.RestPort),
	}
	err := srv.ListenAndServe()
	if err != nil {
		c.logger.Error("%v", err)
		c.shutdown()
	}
}

func (c *Coordinator) setupStore() {
	file := c.config.History.FileName
	os.Remove(file)
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		c.logger.Error("%v", err)
		c.shutdown()
	}

	var cols string
	for i := 0; i < len(c.agentsInfo); i++ {
		agent := c.agentsInfo["agent"+strconv.Itoa(i)]
		cols += fmt.Sprint("agent", agent.index)
		cols += " text"
		if agent.index < (len(c.agentsInfo) - 1) {
			cols += ", "
		}
	}

	sqlStmt := fmt.Sprintf("create table CaptureResults (opaque_streamId text not null, timestamp integer, %v); delete from CaptureResults;", cols)
	_, err = db.Exec(sqlStmt)
	if err != nil {
		c.logger.Error("%q: %s\n", err, sqlStmt)
		c.shutdown()
	}

	var fieldStr, argsStr string
	for i := 0; i < len(c.agentsInfo); i++ {
		agent := c.agentsInfo["agent"+strconv.Itoa(i)]
		fieldStr += fmt.Sprint("agent", agent.index)
		argsStr += "?"
		if agent.index < (len(c.agentsInfo) - 1) {
			fieldStr += ", "
			argsStr += ", "
		}
	}

	statementStr := fmt.Sprintf("insert into CaptureResults(opaque_streamId, timestamp, %v) values(?, ?, %v)", fieldStr, argsStr)
	c.insertStatementStr = statementStr
	c.db = db
}

func (c *Coordinator) storeFlusher() {
	currentTime := time.Now().UnixNano() / int64(time.Millisecond)
	maxHistoryTime := currentTime + int64(c.config.History.Period*60)
	for {
		if currentTime < maxHistoryTime {
			time.Sleep(time.Second * time.Duration(maxHistoryTime-currentTime))
		}
		sqlStmt := `delete from CaptureResults;`
		_, err := c.db.Exec(sqlStmt)
		if err != nil {
			c.logger.Error("Cannot execute %q: %s\n", err, sqlStmt)
			c.shutdown()
		}

	}
}

func connectToAgent(hostName string) (*grpc.ClientConn, error) {
	return grpc.Dial(hostName, grpc.WithInsecure())
}

func (c *Coordinator) ConnectToAgents() {
	for ii, hostName := range c.config.Agents {

		conn, err := connectToAgent(hostName)
		if err != nil {
			c.logger.Error("Unable to connect to the Agent %v %v", err, hostName)
			os.Exit(1)
		}
		c.logger.Info("Connected to the agent %v", hostName)
		c.agentsInfo["agent"+strconv.Itoa(ii)] = &AgentInfo{
			index:    ii,
			hostname: hostName,
			conn:     conn,
			client:   pb.NewAgentServiceClient(conn),
		}
	}
}

func (c *Coordinator) startCapture(wg *sync.WaitGroup, agentInfo *AgentInfo) {
	_, err := agentInfo.client.CaptureSignal(context.Background(), &pb.CoordinatorCaptureRequest{})
	if err != nil {
		c.logger.Error("Unable to start capture on agent %s due to %v", agentInfo.hostname, err)
		c.shutdown()
	}
	wg.Done()
}

func (c *Coordinator) StartCapture() {
	wg := sync.WaitGroup{}
	wg.Add(len(c.agentsInfo))
	for _, agent := range c.agentsInfo {
		go c.startCapture(&wg, agent)
	}
	wg.Wait()
}

func (c *Coordinator) mergeAndStore() {
	tx, err := c.db.Begin()
	if err != nil {
		c.logger.Error("Unable start tx in store %v", err)
		c.shutdown()
	}

	stmt, err := tx.Prepare(c.insertStatementStr)

	if err != nil {
		c.logger.Error("Unable to prepare statement for store %v %v", c.insertStatementStr, err)
		c.shutdown()
	}

	var agentsInfo []*AgentInfo
	for _, agentInfo := range c.agentsInfo {
		agentsInfo = append(agentsInfo, agentInfo)
	}
	timestamp := time.Now().Unix() * 1000

	for rowKey, row := range agentsInfo[0].results {
		lat, _ := strconv.ParseInt(row.Oplatency, 10, 64)
		c.histogram.RecordValue(lat)
		var args []interface{}
		args = append(args, rowKey)
		args = append(args, timestamp)
		args = append(args, row.Oplatency)
		foundInOtherAgents := true

		if len(agentsInfo) > 0 {
			foundInOtherAgents = true
		}
		for i := 1; i < len(agentsInfo); i++ {
			agent := agentsInfo[i]
			if row := agent.results[rowKey]; row != nil {
				args = append(args, row.Oplatency)
				lat, _ := strconv.ParseInt(row.Oplatency, 10, 64)
				c.histogram.RecordValue(lat)
				foundInOtherAgents = true
			}
		}
		if !foundInOtherAgents {
			break
		}

		_, err = stmt.Exec(args...)
		if err != nil {
			c.logger.Error("Error executing insert %v", err)
			c.shutdown()
		}
	}

	for _, agentInfo := range agentsInfo {
		agentInfo.results = nil
	}

	tx.Commit()
}

func (c *Coordinator) getFullCaptureFromDb() (string, error) {
	tx, err := c.db.Begin()
	if err != nil {
		c.logger.Error("Error on db tx %v", err)
		os.Exit(1)
	}
	c.logger.Debug("Executing select query")
	rows, err := tx.Query("select * from CaptureResults;")
	defer rows.Close()
	if err != nil {
		c.logger.Error("Error executing select statement %v", err)
		os.Exit(1)
	}
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	count := len(columns)
	tableData := make([]map[string]interface{}, 0)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		tableData = append(tableData, entry)
	}
	tx.Commit()

	jsonData, err := json.Marshal(tableData)
	if err != nil {

		return "", err
	}
	return string(jsonData), nil
}

func (c *Coordinator) getMaxLatency() int64 {
	return c.histogram.Max()
}

func (c *Coordinator) getResults(wg *sync.WaitGroup, agentInfo *AgentInfo) {
	if response, err := agentInfo.client.AgentResults(context.Background(), &pb.CoordinatorResultsRequest{}); err != nil {
		c.logger.Error("Unable to get results from agent %s due to %v", agentInfo.hostname, err)
		c.shutdown()
	} else {
		c.logger.Info("Got %v capture results from %v", len(response.CaptureMap), agentInfo.hostname)
		agentInfo.results = response.CaptureMap
	}
	wg.Done()
}

func (c *Coordinator) GetResults() {
	wg := sync.WaitGroup{}
	wg.Add(len(c.agentsInfo))
	for _, agent := range c.agentsInfo {
		go c.getResults(&wg, agent)
	}
	wg.Wait()
}

func (c *Coordinator) sayGoodBye(wg *sync.WaitGroup, agentInfo *AgentInfo) {
	_, err := agentInfo.client.AgentResults(context.Background(), &pb.CoordinatorResultsRequest{})
	if err != nil {
		c.logger.Error("Unable to get results from agent %s due to %v", agentInfo.hostname, err)
		os.Exit(1)
	}
	wg.Done()
}

func (c *Coordinator) sayGoodbye(wg *sync.WaitGroup, agentInfo *AgentInfo) {
	c.logger.Info("Saying goodbye to %v", agentInfo.hostname)
	_, err := agentInfo.client.GoodByeSignal(context.Background(), &pb.CoordinatorGoodByeRequest{})
	if err != nil {
		c.logger.Error("Unable to say good bye to agent %s due to %v", agentInfo.hostname, err)
		os.Exit(1)
	}
	wg.Done()
}

func (c *Coordinator) shutdown() {
	wg := sync.WaitGroup{}
	wg.Add(len(c.agentsInfo))
	for _, agent := range c.agentsInfo {
		go c.sayGoodbye(&wg, agent)
	}
	wg.Wait()
	os.Exit(1)
}

func (c *Coordinator) cleanupOnTermination() {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		c.shutdown()
	}()
}

func (c *Coordinator) Run() {
	c.ConnectToAgents()
	c.setupStore()
	go c.startRestServer()
	go c.storeFlusher()
	go c.cleanupOnTermination()

	for {
		c.StartCapture()
		time.Sleep(time.Duration(c.config.Capture.Period) * time.Millisecond)
		c.GetResults()
		go c.mergeAndStore()
		time.Sleep(time.Duration(c.config.Capture.Interval) * time.Millisecond)
	}

}
