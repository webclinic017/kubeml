package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/diegostock12/thesis/ml/pkg/api"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
)

type (
	Scheduler struct {
		logger *zap.Logger

		// TODO see how we'll handle multiple requests coming from multiple parameter servers
		// TODO how to decide how many functions each PS will invoke (we need metrics for this, start with constant)
		// Schedule requests coming from the API
		apiChan chan *api.TrainRequest
		psChan  chan *api.ScheduleRequest

		// TODO might need some kind of map to hold all the different running tasks and also metrics
		// Index to keep the information of the parameter servers

		// TODO add a task queue instead of a channel to handle all the task requests
		// Copy this queue from the kubernetes scheduler
	}
)

const (
	submitTaskURL = "start"
)

// sendRequest sends a request to the Parameter Server
func sendRequest(req *api.TrainRequest) error {
	task := api.TrainTask{
		Parameters:  *req,
		Parallelism: api.DEBUG_PARALLELISM,
	}

	// send request
	body, err := json.Marshal(task)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("%s:%d", api.DEBUG_URL, api.PS_DEBUG_PORT) + "/" + submitTaskURL
	resp, err := http.Post(reqURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.Errorf("Error during operation, code was %d", resp.StatusCode)
	}

	// Get the response
	jobId, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("got response")
	fmt.Println(string(jobId))

	return nil
}

// Periodically consuming metrics to make scheduling decisions
func (s *Scheduler) consumeMetrics() {
	// TODO unimplemented
	s.logger.Info("Scheduler starting to observe metrics")
	select {}
}

// Listen for the messages from the API and schedule the job
// The scheduler here has to
//1) Create parameter server for the task with a new id
//2) Indicate to the parameter server the right amount of functions to invoke
//(It is gonna be a constant of 3 for the start)
func (s *Scheduler) satisfyAPIRequests() {
	s.logger.Info("Scheduler started satisfying the requests from the API")

	for {

		// receive requests from the channel coming from the API
		req := <-s.apiChan

		// Right now for testing just print that we got the request and start a parameter server
		s.logger.Info("Received request to schedule network", zap.String("model", req.ModelType))

		//  TODO this parallelism should be optimized
		// Create a parameter server and start it in a new goroutine
		err := sendRequest(req)
		if err != nil {
			s.logger.Error("Error sending request",
				zap.Error(err))
		}

	}

}

// Loop satisfying the requests coming from the Parameter servers
// PS will send a request to the scheduler at the end of an epoch
// to get the number of functions that should be run in the next iteration
func (s *Scheduler) satisfyPSRequests() {
	s.logger.Info("Scheduler started satisfying the requests from the Parameter Servers")

	for {

		// Wait for requests from the PS
		req := <-s.psChan

		s.logger.Debug("Satisfying TrainJob request...")
		s.logger.Info("Received request from PS", zap.String("psID", req.JobId))

		// TODO this should pack all the intelligence in terms of scheduling requests
		// For now just answer with the same parallelism as before

		s.sendJobResponse(req.Parallelism, req.JobId)
		// TODO answer to the PS through rest

	}
}

// Start all the needed goroutines for
//1) periodically consume the metrics that will be used for taking decisions
//2) get the requests from the API through a channel and start the parameter server on demand
//3) Start the API so the functions can notify about status
func Start(logger *zap.Logger, port int) {

	// Create the scheduler
	s := &Scheduler{
		logger:  logger.Named("scheduler"),
		apiChan: make(chan *api.TrainRequest),
		psChan:  make(chan *api.ScheduleRequest),
	}

	// Start consuming metrics and also listening for requests
	go s.consumeMetrics()
	go s.satisfyAPIRequests()
	go s.satisfyPSRequests()

	// Finally start the API
	go s.Serve(port)

	// Sleep for a couple of second
	s.logger.Debug("sleeping")
	time.Sleep(2 * time.Second)

	// Send a request into our own channel so we can create a PS
	s.logger.Debug("Sending random trainrequest")
	s.apiChan <- &api.TrainRequest{
		ModelType:    "resnet",
		BatchSize:    128,
		Epochs:       3,
		Dataset:      "MNIST",
		LearningRate: 0.01,
		FunctionName: "network",
	}

}
