// Command fiscalagent files receipts with a cash register that nothing else can
// reach.
//
// # Why it exists
//
// The registered virtual cash register is a program on a PC inside the
// restaurant, listening on the local network with no authentication. Our server
// cannot route to it. The till screen in a browser can — but only if three
// browser rules line up, and we control none of them: mixed content, private
// network access, and whether the register answers CORS at all.
//
// This program depends on none of them. It runs on the register's own PC,
// connects **outwards** to our server, asks for receipts to file, files them
// against localhost and reports back. Nothing has to be opened on the
// restaurant's network and nothing has to be configured in a browser.
//
// # Running it
//
//	fiscalagent -server https://restoran.example.uz/api/v1 -token <from the panel>
//
// The register's address defaults to what the panel has stored for the branch,
// which the server puts into each job — this program is told where to call, it
// does not decide.
//
// ⚠️ **It holds no business logic and must never grow any.** The document is
// built on the server from the order; this carries it one hop and brings the
// reply back verbatim. Every temptation to be clever here — retrying on a body
// that "looks like" an error, skipping a filing that seems duplicated — moves a
// piece of tax handling onto an unattended PC in a restaurant, where nobody
// will ever read its log.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"restaurant-backend/internal/agent"
)

func main() {
	server := flag.String("server", "", "Keel API base, e.g. https://restoran.example.uz/api/v1")
	token := flag.String("token", "", "agent token from the panel (Settings → Fiscal register)")
	verbose := flag.Bool("v", false, "log every filing, not just failures")
	flag.Parse()

	if *server == "" {
		*server = os.Getenv("KEEL_SERVER")
	}
	if *token == "" {
		*token = os.Getenv("KEEL_AGENT_TOKEN")
	}
	if *server == "" || *token == "" {
		log.Fatal("-server va -token kerak (yoki KEEL_SERVER / KEEL_AGENT_TOKEN)")
	}
	base := strings.TrimRight(*server, "/")

	// ⚠️ Ctrl-C and the Windows service stop must actually stop it. Without this
	// the poll's own long timeout keeps the process alive for another half
	// minute after every restart, which during an update looks like two agents
	// running at once.
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("fiskal agent ishga tushdi: %s", base)
	agent.Run(ctx, agent.Config{Base: base, Token: *token, Verbose: *verbose})
}
