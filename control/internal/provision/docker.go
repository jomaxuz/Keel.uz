// Package provision starts and stops the container that serves one customer.
//
// It talks to the Docker Engine API directly over its unix socket rather than
// through the official SDK: the whole surface used here is five endpoints of
// plain HTTP and JSON, and the SDK would pull a large dependency tree into a
// service whose entire job is to be small and boring.
//
// Design notes that are not obvious from the code:
//
//   - **No published ports.** Containers join a docker network and are reached
//     by name (`keel-<slug>:8080`). Publishing a port per tenant would mean
//     allocating and tracking ports, and every one of them would be a way into
//     a customer's database from outside.
//
//   - **Each tenant gets its own JWT secret.** A token minted for one
//     restaurant is then not merely unauthorized at another — it is
//     unreadable, which is a different and better kind of no.
//
//   - **Failure never destroys anything.** Ensure creates what is missing and
//     starts what is stopped; it does not remove and recreate on every call,
//     because the uploads volume and the running kitchen behind it are not
//     ours to reset.
package provision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config is everything the control plane knows about how to run a tenant.
type Config struct {
	// Path to the Docker socket, mounted into this container read-write.
	Socket string
	// The tenant server image, with a tag. Changing it and re-provisioning is
	// how a rolling update happens.
	Image string
	// The docker network shared by Caddy, the frontend and every tenant.
	Network string
	// What the tenant server should use to reach Mongo — a hostname on that
	// same network, not the control plane's own URI.
	MongoURI string
	// Host directory holding per-tenant uploads: <root>/<slug>/uploads.
	UploadsRoot string
	// Extra environment every tenant needs and none of them differs on: SMS
	// credentials, provider selection.
	CommonEnv map[string]string
	TZ        string
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	if cfg.Socket == "" {
		cfg.Socket = "/var/run/docker.sock"
	}
	if cfg.TZ == "" {
		cfg.TZ = "Asia/Tashkent"
	}
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", cfg.Socket)
				},
			},
		},
	}
}

// ContainerName is the name a tenant's container always has. Derived from the
// slug rather than stored, so the two can never drift apart.
func ContainerName(slug string) string { return "keel-" + slug }

// Spec is what one tenant's container needs to exist.
type Spec struct {
	Slug      string
	DBName    string
	JWTSecret string
	// Which template the tenant's first brand is created from. ⚠️ Read once, on
	// a tenant that has never booted — see the note where it is sent.
	BusinessType string
	// What the customer calls itself, for the profile the tenant seeds on its
	// very first boot.
	//
	// ⚠️ **The console already knows this and the tenant was guessing.** Without
	// it every new install named itself "My Restaurant" — the installer's
	// placeholder — and a shop was a shop called My Restaurant, on its own site,
	// on its receipts and in its Telegram messages until somebody noticed. Read
	// once, like the business type below it.
	BrandName     string
	AdminUsername string
	AdminPassword string
	// The first of the tenant's domains, used for absolute URLs it generates.
	PrimaryDomain string
	// How the tenant reaches the control plane, and the token that proves
	// which tenant it is. Used for one thing so far: the owner connecting
	// their own domain from their own settings page, without us in the middle.
	//
	// The token is derived from the slug rather than stored, so a container
	// gets its credential simply by being created and there is nothing to keep
	// in sync — the same shape as the kiosk codes in the tenant app.
	ControlURL   string
	ControlToken string
}

// State is what Docker says about a container.
type State struct {
	// "running" | "restarting" | "stopped" | "absent"
	Status string `json:"status"`
	// Non-zero when the process died. Meaningless while running.
	ExitCode int `json:"exitCode,omitempty"`
	// Docker's own wording, for the operator who wants the detail.
	Raw string `json:"raw,omitempty"`
	// The **image this container is actually running**, by id.
	//
	// Not the tag: the tag is a label that moves, and a container keeps
	// running whatever image it was created from long after the tag points
	// somewhere else. That gap is the whole reason a deploy can be green and
	// false at the same time — every health check passes, every commit is
	// right, and the new code is nowhere. Comparing this against ImageID()
	// is the only honest answer to "is this tenant up to date?".
	ImageID string `json:"imageId,omitempty"`
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(raw)
	}
	// The host is ignored for a unix socket but the URL must still parse.
	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, rdr)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("docker: ulanib bo'lmadi (%s): %w", c.cfg.Socket, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if res.StatusCode >= 400 {
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &e)
		msg := e.Message
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		return res.StatusCode, fmt.Errorf("docker %s: %s", path, msg)
	}
	if out != nil && len(raw) > 0 {
		return res.StatusCode, json.Unmarshal(raw, out)
	}
	return res.StatusCode, nil
}

// Status reports whether a tenant's container exists and is running.
func (c *Client) Status(ctx context.Context, slug string) (State, error) {
	var out struct {
		// Docker's own field name for the image id the container runs.
		Image string `json:"Image"`
		State struct {
			Running    bool   `json:"Running"`
			Restarting bool   `json:"Restarting"`
			ExitCode   int    `json:"ExitCode"`
			Status     string `json:"Status"`
			Error      string `json:"Error"`
		} `json:"State"`
	}
	code, err := c.do(ctx, http.MethodGet,
		"/containers/"+url.PathEscape(ContainerName(slug))+"/json", nil, &out)
	if code == http.StatusNotFound {
		return State{Status: "absent"}, nil
	}
	if err != nil {
		return State{}, err
	}
	st := State{Status: "stopped", Raw: out.State.Status, ImageID: out.Image}
	if out.State.Running {
		st.Status = "running"
	}
	// A crash-looping container reports Running between restarts, which is how
	// a broken tenant passes for a healthy one.
	if out.State.Restarting {
		st.Status = "restarting"
	}
	st.ExitCode = out.State.ExitCode
	if out.State.Error != "" {
		st.Raw = out.State.Status + " · " + out.State.Error
	}
	return st, nil
}

// States is every tenant container's state, keyed by slug, in one round trip.
//
// One call rather than one per tenant, for the same reason the period totals
// are one aggregate: asking per customer turns the customer list into N round
// trips and grows its load time with every sale. Here it is worse than with
// Mongo — each one is a container inspect, and the list is rendered on the
// screen somebody keeps open.
//
// A slug missing from the returned map has **no container**, which is a real
// answer and the one this exists to give.
func (c *Client) States(ctx context.Context) (map[string]State, error) {
	var out []struct {
		Names []string `json:"Names"`
		Image string   `json:"ImageID"`
		State string   `json:"State"`
		// Docker's human wording: "Up 3 hours", "Exited (1) 5 minutes ago".
		Status string `json:"Status"`
	}
	// all=1 so a stopped container is reported as stopped rather than as
	// absent — "somebody stopped it" and "it was never created" are different
	// problems with different fixes.
	if _, err := c.do(ctx, http.MethodGet, "/containers/json?all=1", nil, &out); err != nil {
		return nil, err
	}
	states := make(map[string]State, len(out))
	for _, ct := range out {
		for _, n := range ct.Names {
			// Docker returns names with a leading slash.
			name := strings.TrimPrefix(n, "/")
			slug, ok := strings.CutPrefix(name, "keel-")
			if !ok {
				continue
			}
			st := State{Status: "stopped", Raw: ct.Status, ImageID: ct.Image}
			switch ct.State {
			case "running":
				st.Status = "running"
			case "restarting":
				// Reports as running between restarts, which is how a broken
				// tenant passes for a healthy one.
				st.Status = "restarting"
			}
			states[slug] = st
		}
	}
	return states, nil
}

// Ensure brings the container to a running state, creating it if needed.
func (c *Client) Ensure(ctx context.Context, s Spec) error {
	st, err := c.Status(ctx, s.Slug)
	if err != nil {
		return err
	}
	if st.Status == "absent" {
		if err := c.create(ctx, s); err != nil {
			return err
		}
	}
	if st.Status == "running" {
		return nil
	}
	code, err := c.do(ctx, http.MethodPost,
		"/containers/"+url.PathEscape(ContainerName(s.Slug))+"/start", nil, nil)
	// 304 is "already started" — a race with another provision, not a failure.
	if code == http.StatusNotModified {
		return nil
	}
	return err
}

func (c *Client) create(ctx context.Context, s Spec) error {
	env := map[string]string{
		"TZ":        c.cfg.TZ,
		"PORT":      "8080",
		"MONGO_URI": c.cfg.MongoURI,
		// The one line that separates this customer from every other.
		"MONGO_DB":   s.DBName,
		"JWT_SECRET": s.JWTSecret,
		"UPLOAD_DIR": "/app/uploads",
	}
	for k, v := range c.cfg.CommonEnv {
		if v != "" {
			env[k] = v
		}
	}
	if s.PrimaryDomain != "" {
		env["PUBLIC_BASE_URL"] = "https://" + s.PrimaryDomain
		env["CORS_ORIGINS"] = "https://" + s.PrimaryDomain
	}
	if s.ControlURL != "" && s.ControlToken != "" {
		env["CONTROL_URL"] = s.ControlURL
		env["CONTROL_TOKEN"] = s.ControlToken
		env["TENANT_SLUG"] = s.Slug
	}
	// Only sent while the tenant has never booted. Once its owner exists, the
	// tenant server ignores these, and the control plane forgets the password.
	if s.AdminUsername != "" && s.AdminPassword != "" {
		env["ADMIN_USERNAME"] = s.AdminUsername
		env["ADMIN_PASSWORD"] = s.AdminPassword
	}
	// ⚠️ **Sent every time and read once.** The tenant applies it only while it
	// is creating its first brand; on every boot afterwards a brand exists and
	// the value is ignored. Sending it always is what makes a re-provision of a
	// brand-new tenant behave the same as its first boot — and re-provisioning
	// before the owner has logged in is an ordinary thing to do.
	if s.BusinessType != "" {
		env["BUSINESS_TYPE"] = s.BusinessType
	}
	// Same rule, same reason: applied while the profile is being created and
	// ignored on every boot after that. A tenant renamed in the console keeps
	// the name its owner typed into their own settings — which is the one they
	// have been reading on their receipts.
	if s.BrandName != "" {
		env["BRAND_NAME"] = s.BrandName
	}

	list := make([]string, 0, len(env))
	for k, v := range env {
		list = append(list, k+"="+v)
	}

	body := map[string]any{
		"Image": c.cfg.Image,
		"Env":   list,
		"Labels": map[string]string{
			"keel.tenant": s.Slug,
		},
		"HostConfig": map[string]any{
			// Restarts with the host. A restaurant's site coming back after a
			// reboot should not require anybody to notice it went away.
			"RestartPolicy": map[string]any{"Name": "unless-stopped"},
			"NetworkMode":   c.cfg.Network,
			"Binds": []string{
				strings.TrimRight(c.cfg.UploadsRoot, "/") + "/" + s.Slug + "/uploads:/app/uploads",
			},
			// A noisy tenant must not be able to take the others down with it.
			//
			// The ceiling: 512 MB and one core. A tenant server idles at about
			// 9 MB, so this is not a budget anybody is expected to spend — it is
			// the wall a runaway hits before the host notices.
			"Memory":   int64(512) << 20,
			"NanoCpus": int64(1_000_000_000),
			// ⚠️ **No swap**, and this is the line that protects the neighbours
			// rather than the tenant. Docker's default is twice `Memory`, so a
			// container leaking memory would quietly spend 512 MB of *disk* as
			// slow memory — on the one disk Mongo, every other tenant's uploads
			// and the nightly backup all share. The failure that follows is not
			// "one site is down", it is "the whole box got slow", which is far
			// harder to trace back. A container that hits its ceiling should die
			// and be restarted, loudly and locally.
			"MemorySwap": int64(512) << 20,
			// ⚠️ A fork bomb, a goroutine leak spawning threads, or a wedged
			// process pool exhausts the **host's** pid space, not the
			// container's — and the host running out of pids means nothing else
			// can start either, including the tools somebody would use to fix
			// it. 512 is ~50× what a Go server with a few dozen goroutines uses.
			"PidsLimit": 512,
			// Relative weight when the CPU is actually contended. NanoCpus is the
			// hard ceiling and says nothing about *sharing*: with only ceilings,
			// three tenants wanting a core each at lunchtime are resolved by the
			// scheduler's own defaults. Equal shares make that fair by
			// construction, and give the control plane and Mongo — which have no
			// share set and so keep the default 1024 — no less than a tenant.
			"CpuShares": 1024,
			// Disk fairness, same idea: one tenant restoring a large image set
			// must not stall everybody else's reads.
			"BlkioWeight": 500,
			// ⚠️ **Docker's default is 1024 open files, and that is a ceiling
			// on concurrent visitors, not on anything a tenant does wrong.**
			// Every accepted connection is a descriptor, and so is every
			// connection in the Mongo pool; the load test of 2026-09-03 ran the
			// server at ~350 req/s with `ulimit -n` still at 1024, which is
			// close enough to matter. It fails the worst way there is: accept()
			// starts returning "too many open files" at exactly the busiest
			// minute of the day, and recovers by itself the moment the rush
			// ends — so by the time anybody looks, the site is fine.
			//
			// A limit, still: 64000 is the same ceiling Mongo runs with, far
			// above any honest load, and low enough that a descriptor leak is
			// stopped before it reaches the host's own limit.
			"Ulimits": []map[string]any{
				{"Name": "nofile", "Soft": 64000, "Hard": 64000},
			},
		},
	}
	_, err := c.do(ctx, http.MethodPost,
		"/containers/create?name="+url.QueryEscape(ContainerName(s.Slug)), body, nil)
	return err
}

// Stop halts a suspended tenant, freeing its memory. The data is untouched:
// a customer who pays on Thursday gets Wednesday's menu back.
func (c *Client) Stop(ctx context.Context, slug string) error {
	code, err := c.do(ctx, http.MethodPost,
		"/containers/"+url.PathEscape(ContainerName(slug))+"/stop?t=10", nil, nil)
	// Already stopped, or never existed — both are the state we wanted.
	if code == http.StatusNotModified || code == http.StatusNotFound {
		return nil
	}
	return err
}

// Remove deletes the container. Deliberately not the volume: uploads are the
// customer's photographs, and this is called on re-provision as well as on
// departure.
func (c *Client) Remove(ctx context.Context, slug string) error {
	code, err := c.do(ctx, http.MethodDelete,
		"/containers/"+url.PathEscape(ContainerName(slug))+"?force=true", nil, nil)
	if code == http.StatusNotFound {
		return nil
	}
	return err
}

// Image is the tag tenants are started from.
func (c *Client) Image() string { return c.cfg.Image }

// ImageID resolves the configured tag to the image it currently names.
//
// The tag is a moving label. `keel-tenant:latest` after a deploy is a
// different image than it was an hour ago, while every container created from
// the old one keeps running the old code — and reports itself perfectly
// healthy while doing so. Rolling out means walking the tenants whose
// container image id is not this one.
func (c *Client) ImageID(ctx context.Context) (string, error) {
	var out struct {
		ID string `json:"Id"`
	}
	code, err := c.do(ctx, http.MethodGet,
		"/images/"+url.PathEscape(c.cfg.Image)+"/json", nil, &out)
	if code == http.StatusNotFound {
		return "", fmt.Errorf("image topilmadi: %s", c.cfg.Image)
	}
	if err != nil {
		return "", err
	}
	return out.ID, nil
}

// Recreate replaces a container with one built from the current image and
// spec — the rolling-update path. Done one tenant at a time by the caller:
// fifty containers restarting together would hit Mongo with fifty migrations
// at once.
func (c *Client) Recreate(ctx context.Context, s Spec) error {
	if err := c.Remove(ctx, s.Slug); err != nil {
		return err
	}
	return c.Ensure(ctx, s)
}

// DiskUsage is what Docker itself is holding on this host.
//
// Reported beside the filesystem's own figures because on this box they are
// nearly the same number, and this one says *what* is using it: images left
// behind by deploys, stopped containers, and the volumes holding every
// customer's photographs. The first two are reclaimable in one command; the
// third is not, and knowing which is which is the whole point.
type DiskUsage struct {
	Images     int64 `json:"images"`
	Containers int64 `json:"containers"`
	Volumes    int64 `json:"volumes"`
	BuildCache int64 `json:"buildCache"`
	// What `docker system prune` would actually free.
	Reclaimable int64 `json:"reclaimable"`
}

func (c *Client) DiskUsage(ctx context.Context) (DiskUsage, error) {
	var out struct {
		LayersSize int64 `json:"LayersSize"`
		Images     []struct {
			Size       int64 `json:"Size"`
			Containers int   `json:"Containers"`
		} `json:"Images"`
		Containers []struct {
			SizeRw int64 `json:"SizeRw"`
		} `json:"Containers"`
		Volumes []struct {
			UsageData struct {
				Size int64 `json:"Size"`
			} `json:"UsageData"`
		} `json:"Volumes"`
		BuildCache []struct {
			Size  int64 `json:"Size"`
			InUse bool  `json:"InUse"`
		} `json:"BuildCache"`
	}
	if _, err := c.do(ctx, http.MethodGet, "/system/df", nil, &out); err != nil {
		return DiskUsage{}, err
	}
	du := DiskUsage{Images: out.LayersSize}
	for _, i := range out.Images {
		// An image no container uses is one a deploy left behind.
		if i.Containers == 0 {
			du.Reclaimable += i.Size
		}
	}
	for _, ct := range out.Containers {
		du.Containers += ct.SizeRw
	}
	for _, v := range out.Volumes {
		du.Volumes += v.UsageData.Size
	}
	for _, b := range out.BuildCache {
		du.BuildCache += b.Size
		if !b.InUse {
			du.Reclaimable += b.Size
		}
	}
	return du, nil
}

// Ping proves the socket is reachable and says which Docker answered.
func (c *Client) Ping(ctx context.Context) (string, error) {
	var out struct {
		Version    string `json:"Version"`
		APIVersion string `json:"ApiVersion"`
	}
	if _, err := c.do(ctx, http.MethodGet, "/version", nil, &out); err != nil {
		return "", err
	}
	return "Docker " + out.Version + " (API " + out.APIVersion + ")", nil
}

// Logs returns the tail of a container's output.
//
// Shown to the operator when provisioning fails, because the useful sentence
// is almost never ours: it is the tenant server saying it cannot reach Mongo,
// or that a required variable is missing.
func (c *Client) Logs(ctx context.Context, slug string, lines int) string {
	if lines <= 0 {
		lines = 20
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("http://docker/containers/%s/logs?stdout=1&stderr=1&tail=%d",
			url.PathEscape(ContainerName(slug)), lines), nil)
	if err != nil {
		return ""
	}
	res, err := c.http.Do(req)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return strings.TrimSpace(demux(raw))
}

// demux strips Docker's 8-byte stream framing. Without it the log arrives with
// a control byte and three nulls in front of every line.
func demux(raw []byte) string {
	var b strings.Builder
	for len(raw) >= 8 {
		n := int(raw[4])<<24 | int(raw[5])<<16 | int(raw[6])<<8 | int(raw[7])
		if n < 0 || n > len(raw)-8 {
			// Not framed (a TTY container writes plain bytes).
			return string(raw)
		}
		b.Write(raw[8 : 8+n])
		raw = raw[8+n:]
	}
	return b.String()
}

// IP returns the container's address on our network.
//
// Used instead of its name because the control plane may be reached from the
// host during development, where docker's DNS does not resolve. The address is
// asked for fresh every time: it changes when a container is recreated.
func (c *Client) IP(ctx context.Context, slug string) (string, error) {
	var out struct {
		NetworkSettings struct {
			Networks map[string]struct {
				IPAddress string `json:"IPAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}
	if _, err := c.do(ctx, http.MethodGet,
		"/containers/"+url.PathEscape(ContainerName(slug))+"/json", nil, &out); err != nil {
		return "", err
	}
	if n, ok := out.NetworkSettings.Networks[c.cfg.Network]; ok && n.IPAddress != "" {
		return n.IPAddress, nil
	}
	for _, n := range out.NetworkSettings.Networks {
		if n.IPAddress != "" {
			return n.IPAddress, nil
		}
	}
	return "", fmt.Errorf("konteynerning tarmoq manzili yo'q")
}

// WaitHealthy waits for the tenant to actually answer.
//
// Not "is the process alive": a server that cannot reach its database stays
// alive for the whole of its connection timeout, and a liveness check happily
// calls that ready. It then gets recorded as provisioned, its admin password is
// discarded, and nobody looks at it again until the customer phones. The only
// honest signal is the tenant answering its own health endpoint — the same
// distinction the iiko adapter draws between "sent" and "accepted".
func (c *Client) WaitHealthy(ctx context.Context, slug string, d time.Duration) error {
	deadline := time.Now().Add(d)
	probe := &http.Client{Timeout: 3 * time.Second}
	var last string

	for {
		st, err := c.Status(ctx, slug)
		if err != nil {
			return err
		}
		switch st.Status {
		case "restarting":
			return fmt.Errorf("konteyner qayta-qayta o'chib yonmoqda:\n%s", c.Logs(ctx, slug, 15))
		case "absent":
			return fmt.Errorf("konteyner topilmadi")
		case "stopped":
			return fmt.Errorf("konteyner %d kodi bilan to'xtadi:\n%s",
				st.ExitCode, c.Logs(ctx, slug, 15))
		}

		if ip, err := c.IP(ctx, slug); err == nil && ip != "" {
			res, err := probe.Get("http://" + ip + ":8080/health")
			if err == nil {
				res.Body.Close()
				if res.StatusCode < 400 {
					return nil
				}
				last = res.Status
			} else {
				last = err.Error()
			}
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("konteyner ishga tushdi, lekin javob bermayapti (%s):\n%s",
				last, c.Logs(ctx, slug, 15))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// Prune frees what Docker is holding and nothing else.
//
// ⚠️ **Three calls, not `docker system prune -a`.** The dangerous flag is `-a`,
// which deletes every image no *running* container uses — including the previous
// tenant image, which is the only thing a rollback has to roll back to. And
// `--volumes` would delete the volumes holding every customer's photographs.
//
// So: dangling images (layers no tag points at, which is what a rebuild leaves
// behind), stopped containers, and the build cache. Each is asked for separately
// because each fails separately, and the caller reports what was actually freed
// rather than what was attempted.
func (c *Client) Prune(ctx context.Context) (freed int64, err error) {
	type reply struct {
		SpaceReclaimed int64 `json:"SpaceReclaimed"`
	}
	var firstErr error

	// Dangling only: `filters={"dangling":["true"]}` is the default for this
	// endpoint, and it is the difference between "images a rebuild orphaned" and
	// "every image except the running one".
	var images reply
	if _, e := c.do(ctx, http.MethodPost, "/images/prune", nil, &images); e != nil {
		firstErr = e
	}
	freed += images.SpaceReclaimed

	var containers reply
	if _, e := c.do(ctx, http.MethodPost, "/containers/prune", nil, &containers); e != nil && firstErr == nil {
		firstErr = e
	}
	freed += containers.SpaceReclaimed

	var cache reply
	if _, e := c.do(ctx, http.MethodPost, "/build/prune", nil, &cache); e != nil && firstErr == nil {
		firstErr = e
	}
	freed += cache.SpaceReclaimed

	return freed, firstErr
}

// PurgeUploads erases one tenant's photographs from the host.
//
// ⚠️ **Through a one-shot container, not through a mount of our own**, and that
// is the whole design. The console mounts the uploads root **read-only** so that
// nothing it does can ever write into a customer's files; loosening that mount to
// make one rare button work would trade a standing guarantee for a convenience.
// Docker access is a power the control plane already holds — this spends it once,
// scoped to a single directory, instead of holding a permanent write path.
//
// The bind is the tenant's **own** directory, never the root with a subpath
// appended: a mount of the parent would give the throwaway container write access
// to every other restaurant's photographs for the length of an `rm -rf`, and the
// argument that decides which one is deleted would be a string.
//
// The directory itself is left behind, empty. Removing it would need the parent
// mounted, and an empty folder is harmless — it is also a small piece of evidence
// that this ran.
func (c *Client) PurgeUploads(ctx context.Context, slug string) error {
	if strings.TrimSpace(slug) == "" {
		return errors.New("slug bo'sh")
	}
	dir := strings.TrimRight(c.cfg.UploadsRoot, "/") + "/" + slug
	name := "keel-purge-" + slug

	// Left over from a previous attempt that died mid-way; the name would
	// otherwise be taken and every retry would fail for a reason nobody can see.
	_, _ = c.do(ctx, http.MethodDelete, "/containers/"+url.PathEscape(name)+"?force=true", nil, nil)

	body := map[string]any{
		// The tenant image, because it is the one image this host is guaranteed
		// to have: pulling `alpine` here would make a destructive step depend on
		// the network, and fail exactly when somebody is trying to close a
		// customer's account.
		"Image":      c.cfg.Image,
		"Entrypoint": []string{"/bin/sh", "-c"},
		"Cmd":        []string{"rm -rf /target/* /target/.[!.]* 2>/dev/null; exit 0"},
		"HostConfig": map[string]any{
			"Binds":       []string{dir + ":/target"},
			"AutoRemove":  true,
			"NetworkMode": "none",
			"Memory":      int64(64) << 20,
		},
	}
	if _, err := c.do(ctx, http.MethodPost, "/containers/create?name="+url.QueryEscape(name), body, nil); err != nil {
		return err
	}
	if _, err := c.do(ctx, http.MethodPost, "/containers/"+url.PathEscape(name)+"/start", nil, nil); err != nil {
		return err
	}
	// Waited for rather than fired and forgotten: the caller reports what was
	// actually done, and "the files are gone" is the one claim here that must
	// not be a guess.
	var res struct {
		StatusCode int `json:"StatusCode"`
	}
	if _, err := c.do(ctx, http.MethodPost, "/containers/"+url.PathEscape(name)+"/wait", nil, &res); err != nil {
		return err
	}
	if res.StatusCode != 0 {
		return fmt.Errorf("rasm fayllarini o'chirish %d bilan tugadi", res.StatusCode)
	}
	return nil
}
