package extension

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"extension-scaffold/internal/config"
	"extension-scaffold/pkg/types"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
	teeutils "github.com/flare-foundation/tee-node/pkg/utils"

	"github.com/flare-foundation/tee-node/pkg/processorutils"
)

// --- CUSTOMIZE: Add your extension's state fields here. ---
// These fields hold your extension's in-memory state. They are returned
// by the GET /state endpoint and should reflect the cumulative result
// of all processed actions. Protect access with the mu mutex.

type Extension struct {
	mu     sync.RWMutex
	Server *http.Server

	// TODO: Add your state fields here. For example:
	// orderCount int
	// lastOrder  string
}

// --- DO NOT MODIFY: New(), stateHandler(), actionHandler() are boilerplate. ---

func New(extensionPort, signPort int) *Extension {
	e := &Extension{}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /state", e.stateHandler)
	mux.HandleFunc("POST /action", e.actionHandler)

	e.Server = &http.Server{Addr: fmt.Sprintf(":%d", extensionPort), Handler: mux}
	return e
}

func (e *Extension) stateHandler(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	stateResponse := types.StateResponse{
		StateVersion: teeutils.ToHash(config.Version),
		State: types.State{
			// TODO: Return your state fields here. For example:
			// OrderCount: e.orderCount,
			// LastOrder:  e.lastOrder,
		},
	}
	e.mu.RUnlock()

	err := json.NewEncoder(w).Encode(stateResponse)
	if err != nil {
		http.Error(w, fmt.Sprintf("sending response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (e *Extension) actionHandler(w http.ResponseWriter, r *http.Request) {
	var action teetypes.Action
	err := json.NewDecoder(r.Body).Decode(&action)
	if err != nil {
		http.Error(w, fmt.Sprintf("decoding action: %v", err), http.StatusBadRequest)
		return
	}

	logger.Infof("received action, ID: %s", action.Data.ID)

	status, body := e.processAction(action)

	logger.Infof("sending action result, ID: %s, status: %d, log: %s", action.Data.ID, status, getLogFromBody(body))

	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// --- CUSTOMIZE: processAction() is the routing layer for your extension. ---
//
// This function receives every action from the TEE node and routes it to
// the correct handler based on the OPType field. Add a case for each
// operation type your extension supports.
//
// The OPType is a bytes32 hash. Use teeutils.ToHash("YOUR_OP_TYPE") to
// match against the constant defined in config.go (which must also match
// the bytes32 constant in your Solidity contract).

func (e *Extension) processAction(action teetypes.Action) (int, []byte) {
	dataFixed, err := processorutils.Parse[instruction.DataFixed](action.Data.Message)
	if err != nil {
		return http.StatusBadRequest, []byte(fmt.Sprintf("decoding fixed data: %v", err))
	}

	switch {
	case dataFixed.OPType == teeutils.ToHash(config.OPTypeMyAction):
		ar := e.processMyAction(action, dataFixed)
		b, _ := json.Marshal(ar)
		return http.StatusOK, b

	// TODO: Add more cases for additional operation types. For example:
	// case dataFixed.OPType == teeutils.ToHash(config.OPTypeAnotherAction):
	//     ar := e.processAnotherAction(action, dataFixed)
	//     b, _ := json.Marshal(ar)
	//     return http.StatusOK, b

	default:
		return http.StatusNotImplemented, []byte("unsupported op type")
	}
}

// --- CUSTOMIZE: Implement your action handlers below. ---
//
// Each handler follows the same pattern:
//   1. Decode the incoming message from df.OriginalMessage into your request type
//   2. Validate the request
//   3. Execute your business logic (this is where your extension does its work)
//   4. Build a response and return it via buildResult()
//
// Status codes for buildResult:
//   0 = error (include the error in the err parameter)
//   1 = success (include the response data)

func (e *Extension) processMyAction(action teetypes.Action, df *instruction.DataFixed) teetypes.ActionResult {
	// Step 1: Decode the incoming message.
	var req types.MyActionRequest
	dec := json.NewDecoder(bytes.NewReader(df.OriginalMessage))
	dec.DisallowUnknownFields()
	err := dec.Decode(&req)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
	}

	// Step 2: Validate the request.
	// TODO: Add your validation logic here. For example:
	// if req.Amount == 0 {
	//     return buildResult(action, df, nil, 0, fmt.Errorf("amount must be greater than zero"))
	// }

	// Step 3: Execute your business logic.
	// TODO: This is the core of your extension. Implement your logic here.
	// Examples of what extensions might do:
	//   - Call external APIs or services
	//   - Use the TEE's signing capabilities (via the sign port)
	//   - Perform computations on confidential data
	//   - Manage encrypted state

	// Step 4: Build and return the response.
	resp := types.MyActionResponse{
		// TODO: Populate your response fields here. For example:
		// TxHash: "0x...",
		// Status: "confirmed",
	}
	data, _ := json.Marshal(resp)

	// Update extension state (protected by mutex).
	e.mu.Lock()
	// TODO: Update your state fields here. For example:
	// e.orderCount++
	// e.lastOrder = req.OrderID
	e.mu.Unlock()

	return buildResult(action, df, data, 1, nil)
}

// --- DO NOT MODIFY below this line. ---

func buildResult(a teetypes.Action, df *instruction.DataFixed, data []byte, status uint8, err error) teetypes.ActionResult {
	ar := teetypes.ActionResult{
		ID:            a.Data.ID,
		SubmissionTag: a.Data.SubmissionTag,
		Version:       config.Version,
		OPType:        df.OPType,
		OPCommand:     df.OPCommand,
		Data:          data,
		Status:        status,
	}
	switch status {
	case 0:
		ar.Log = fmt.Sprintf("error: %v", err)
	case 1:
		ar.Log = "ok"
	}
	return ar
}

func getLogFromBody(body []byte) string {
	var ar teetypes.ActionResult
	if err := json.Unmarshal(body, &ar); err != nil {
		return string(body)
	}
	return ar.Log
}
