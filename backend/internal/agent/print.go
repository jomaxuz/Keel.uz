package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"restaurant-backend/internal/printer"
)

// Printing, from the one program that can reach the printer.
//
// ⚠️ **The same argument the fiscal half is built on.** A receipt printer sits
// on the restaurant's own network or on a USB port of the PC by the till; our
// server cannot route to either, and a browser cannot open a socket at all.
// This program is already there and already asking us for work.

// printOne writes one job's bytes to one printer.
func printOne(ctx context.Context, j printJob) error {
	payload, err := base64.StdEncoding.DecodeString(j.Payload)
	if err != nil {
		return fmt.Errorf("chop etish ma'lumoti buzuq: %w", err)
	}
	target, err := printer.Parse(j.Target)
	if err != nil {
		return err
	}
	return printer.Send(ctx, target, payload)
}

// reportPrint tells the server what the printer did.
//
// ⚠️ **Sent even when it worked**, because "printed" is the only thing that
// takes a job off the queue — and a queue that only hears about failures prints
// every kitchen ticket twice after a restart.
func reportPrint(
	ctx context.Context, c *http.Client, base, token, id string, printErr error,
) error {
	body := map[string]string{}
	if printErr != nil {
		body["error"] = printErr.Error()
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		base+"/fiscal/agent/print/"+id, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	if res.StatusCode >= 400 {
		return fmt.Errorf("server javobi: %d", res.StatusCode)
	}
	return nil
}
