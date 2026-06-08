package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"

	"qpi/internal/config"
	"qpi/internal/scheduler"
)

var (
	activeQPUs   = make(map[string]context.CancelFunc) // qpuID -> cancel
	activeQPUsMu sync.Mutex
)

type registerRequest struct {
	Name              string `json:"name"`
	RegistrationToken string `json:"registration_token"`
}

type registerResponse struct {
	Status         string `json:"status"`
	NNGCommandPort int    `json:"nng_command_port"`
	NNGResultPort  int    `json:"nng_result_port"`
	AuthToken      string `json:"auth_token"`
}

type resultPayload struct {
	JobID   string         `json:"job_id"`
	Results map[string]any `json:"results"`
}

func RegisterRoutes(e *core.ServeEvent) {
	e.Router.POST("/api/qpu/register", func(re *core.RequestEvent) error {
		return handleQPURegister(re)
	})
}

func handleQPURegister(re *core.RequestEvent) error {
	var req registerRequest
	if err := re.BindBody(&req); err != nil {
		return re.Error(http.StatusBadRequest, "invalid request body", err)
	}

	// Find QPU by registration token
	qpu, err := re.App.FindFirstRecordByData(config.CollectionQPUs, "registration_token", req.RegistrationToken)
	if err != nil {
		return re.Error(http.StatusUnauthorized, "invalid registration token", err)
	}

	// Update name if provided
	if req.Name != "" {
		qpu.Set("name", req.Name)
	}

	// Allocate ports if not already done
	cmdPort := qpu.GetInt("nng_command_port")
	resPort := qpu.GetInt("nng_result_port")

	if cmdPort == 0 || resPort == 0 {
		ports, err := findFreePorts(re.App, 2)
		if err != nil {
			return re.Error(http.StatusInternalServerError, "cannot allocate NNG ports", err)
		}
		cmdPort = ports[0]
		resPort = ports[1]
		qpu.Set("nng_command_port", cmdPort)
		qpu.Set("nng_result_port", resPort)
	}

	qpu.Set("status", "online")
	if err := re.App.Save(qpu); err != nil {
		return re.Error(http.StatusInternalServerError, "cannot save QPU record", err)
	}

	// Spin up orchestration goroutines if not already running
	qpuID := qpu.Id
	activeQPUsMu.Lock()
	if _, running := activeQPUs[qpuID]; !running {
		ctx, cancel := context.WithCancel(context.Background())
		activeQPUs[qpuID] = cancel
		go runDispatcher(re.App, qpuID, cmdPort)
		go runResultListener(re.App, qpuID, resPort)
		log.Printf("[QPi] Goroutines started for QPU %s (cmd:%d res:%d)", qpuID, cmdPort, resPort)
		_ = ctx // ctx is held for future cancellation via cancel()
	}
	activeQPUsMu.Unlock()

	// Generate auth token for the QPU record
	token, err := qpu.NewStaticAuthToken(0) // 0 = use app default duration
	if err != nil {
		// Non-fatal: fall back to the registration token as a simple identifier
		token = req.RegistrationToken
	}

	return re.JSON(http.StatusOK, registerResponse{
		Status:         "success",
		NNGCommandPort: cmdPort,
		NNGResultPort:  resPort,
		AuthToken:      token,
	})
}

func runDispatcher(app core.App, qpuID string, cmdPort int) {
	addr := fmt.Sprintf("tcp://0.0.0.0:%d", cmdPort)
	sock, err := push.NewSocket()
	if err != nil {
		log.Printf("[Dispatcher %s] socket error: %v", qpuID, err)
		return
	}
	defer sock.Close()

	if err := sock.Listen(addr); err != nil {
		log.Printf("[Dispatcher %s] listen error on %s: %v", qpuID, addr, err)
		return
	}
	log.Printf("[Dispatcher %s] PUSH listening on %s", qpuID, addr)

	for {
		job := scheduler.FetchNextJob(app, qpuID)
		if job == nil {
			time.Sleep(config.DispatchPollInterval)
			continue
		}

		payload, _ := json.Marshal(map[string]any{
			"job_id":  job.Id,
			"payload": job.Get("payload"),
		})

		if err := sock.Send(payload); err != nil {
			log.Printf("[Dispatcher %s] send error: %v — requeueing", qpuID, err)
			job.Set("status", "pending")
			_ = app.Save(job)
			time.Sleep(config.DispatchPollInterval)
			continue
		}

		job.Set("status", "running")
		if err := app.Save(job); err != nil {
			log.Printf("[Dispatcher %s] DB update error: %v", qpuID, err)
		} else {
			log.Printf("[Dispatcher %s] dispatched job %s", qpuID, job.Id)
		}
	}
}

func runResultListener(app core.App, qpuID string, resPort int) {
	addr := fmt.Sprintf("tcp://0.0.0.0:%d", resPort)
	sock, err := pull.NewSocket()
	if err != nil {
		log.Printf("[Listener %s] socket error: %v", qpuID, err)
		return
	}
	defer sock.Close()

	if err := sock.Listen(addr); err != nil {
		log.Printf("[Listener %s] listen error on %s: %v", qpuID, addr, err)
		return
	}
	log.Printf("[Listener %s] PULL listening on %s", qpuID, addr)

	for {
		msg, err := sock.Recv()
		if err != nil {
			if err == mangos.ErrClosed {
				return
			}
			log.Printf("[Listener %s] recv error: %v", qpuID, err)
			time.Sleep(config.DispatchPollInterval)
			continue
		}

		var result resultPayload
		if err := json.Unmarshal(msg, &result); err != nil {
			log.Printf("[Listener %s] JSON parse error: %v", qpuID, err)
			continue
		}

		job, err := app.FindRecordById(config.CollectionQuantumJobs, result.JobID)
		if err != nil {
			log.Printf("[Listener %s] job %s not found: %v", qpuID, result.JobID, err)
			continue
		}

		job.Set("status", "completed")
		job.Set("finished_at", time.Now().UTC().Format("2006-01-02 15:04:05.000Z"))
		resultsJSON, _ := json.Marshal(result.Results)
		job.Set("results", string(resultsJSON))

		if err := app.Save(job); err != nil {
			log.Printf("[Listener %s] DB save error for job %s: %v", qpuID, result.JobID, err)
		} else {
			log.Printf("[Listener %s] job %s completed", qpuID, result.JobID)
		}
	}
}

func findFreePorts(app core.App, count int) ([]int, error) {
	allocated := make(map[int]bool)
	filter := fmt.Sprintf("nng_command_port >= %d || nng_result_port >= %d", config.PortRangeStart, config.PortRangeStart)
	records, err := app.FindRecordsByFilter(config.CollectionQPUs, filter, "", 0, 0)
	if err == nil {
		for _, r := range records {
			cmd := r.GetInt("nng_command_port")
			res := r.GetInt("nng_result_port")
			if cmd > 0 {
				allocated[cmd] = true
			}
			if res > 0 {
				allocated[res] = true
			}
		}
	}

	var ports []int
	for port := config.PortRangeStart; port < config.PortRangeEnd; port++ {
		if allocated[port] {
			continue
		}
		addr := fmt.Sprintf(":%d", port)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			ports = append(ports, port)
			allocated[port] = true
			if len(ports) == count {
				return ports, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find %d free ports in range %d-%d", count, config.PortRangeStart, config.PortRangeEnd)
}
