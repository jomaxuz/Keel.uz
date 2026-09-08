package handlers

// ---- Building a restaurant's own Android app, from the console ----
//
// ⚠️ **A queue of one, and the depth is the point.** Gradle and the Kotlin
// compiler together want more memory than this machine has spare beside every
// customer's container. Two builds at once is a box that swaps, and "the site
// was slow this afternoon" costs far more than a build somebody waited nine
// minutes for. The build script holds a `flock` of its own as well — a lock only
// the caller holds is one a person with a shell walks straight past.
//
// ⚠️ **The artifact is deleted the moment it has been downloaded, and the record
// is not.** Two and a half megabytes per build on a machine that serves every
// customer adds up to a directory nobody prunes; but "which version is on the
// store", asked months later, is a question a filesystem cannot answer. So the
// file goes and the row stays — with the version, the hash, and both names.
//
// ⚠️ **Deleted after the transfer finished, never before it started.** Removing
// the file as the first byte goes out means a dropped connection costs the
// restaurant another nine minutes, and the second attempt finds nothing.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"keel-control/internal/httpx"
	"keel-control/internal/models"
	"keel-control/internal/provision"
)

// How many builds may be waiting. ⚠️ Small on purpose: a queue that accepts
// fifty presses is a queue whose fiftieth build starts seven hours later, by
// which time nobody remembers asking for it. Past this the console says "one is
// already running" — which is the truth and is actionable.
const appBuildQueue = 4

// How long one build may take before it is killed. ⚠️ Generous: nine minutes is
// normal on this machine and a cold Gradle cache is much worse. A build that
// hangs holds the queue, which is what the ceiling is for.
const appBuildTimeout = 40 * time.Minute

// StartAppBuild queues a build of this tenant's app.
func (h *Handler) StartAppBuild(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(r.Context(), bson.M{"_id": id}).Decode(&t); err != nil {
		httpx.Error(w, http.StatusNotFound, "mijoz topilmadi")
		return
	}
	var req struct {
		Format string `json:"format"`
	}
	_ = httpx.Decode(r, &req)
	// ⚠️ **Asked, never assumed.** An APK is what somebody installs on a phone
	// this afternoon; an AAB is what Play accepts and cannot be installed at
	// all. Guessing wrong wastes a nine-minute build and is discovered at the
	// end of an upload.
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format != "aab" {
		format = "apk"
	}

	// ⚠️ **Two separate reasons, and the message says which.** No Docker is a
	// laptop; no image is a server where the toolchain has not been built yet.
	// One sentence covering both sends whoever reads it to the wrong place.
	if h.Docker == nil {
		httpx.Error(w, http.StatusServiceUnavailable,
			"bu o'rnatmada build qilib bo'lmaydi — Docker ulanmagan")
		return
	}
	if h.Cfg.AppBuildImage == "" {
		httpx.Error(w, http.StatusServiceUnavailable,
			"build image sozlanmagan — APP_BUILD_IMAGE ni qo'ying")
		return
	}

	// ⚠️ **One unfinished build per tenant.** Pressing twice is what people do
	// when a button does not visibly change, and the second press would spend
	// another nine minutes producing a byte-identical file.
	busy, _ := h.Store.AppBuilds.CountDocuments(r.Context(), bson.M{
		"tenantId": id,
		"status":   bson.M{"$in": []string{models.AppQueued, models.AppBuilding}},
	})
	if busy > 0 {
		httpx.Error(w, http.StatusConflict, "bu mijoz uchun build allaqachon navbatda")
		return
	}

	build := models.AppBuild{
		TenantID: id, Slug: t.Slug, Format: format,
		Status: models.AppQueued, By: h.pressedBy(r), CreatedAt: time.Now(),
	}
	res, err := h.Store.AppBuilds.InsertOne(r.Context(), build)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		build.ID = oid
	}

	select {
	case h.appBuilds <- build.ID:
	default:
		// ⚠️ Marked failed rather than left queued forever: a row that says
		// "waiting" and is waiting for nothing is the worst of the three states.
		h.finishAppBuild(build.ID, bson.M{
			"status": models.AppFailed,
			"error":  "navbat to'la — biroz kuting va qaytadan urinib ko'ring",
		})
		httpx.Error(w, http.StatusTooManyRequests, "navbat to'la — biroz kuting")
		return
	}
	httpx.JSON(w, http.StatusAccepted, build)
}

// SetAndroidAppID records the Firebase app id this restaurant's app registers
// with.
//
// ⚠️ **Its own endpoint rather than a field on the tenant form.** It belongs
// with the app: an operator editing a customer's name and price has no business
// meeting a Firebase identifier, and the person who has just created one in a
// Firebase console is looking at the app panel.
func (h *Handler) SetAndroidAppID(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var req struct {
		AppID string `json:"appId"`
	}
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	appID := strings.TrimSpace(req.AppID)
	// ⚠️ **Shape-checked, because a wrong one fails silently for ever.** A
	// mistyped id is accepted by Firebase's client, `getToken()` succeeds, and
	// every notification goes nowhere with no error on either side. The format
	// is "1:<sender>:android:<hash>" and anything else is a typo worth catching
	// on the screen it was typed into.
	if appID != "" && !androidAppIDRe.MatchString(appID) {
		httpx.Error(w, http.StatusBadRequest,
			"Firebase app id shakli noto'g'ri — «1:889013622083:android:…» bo'lishi kerak")
		return
	}
	if _, err := h.Store.Tenants.UpdateByID(r.Context(), id,
		bson.M{"$set": bson.M{"androidAppId": appID}}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"appId": appID})
}

// androidAppIDRe is Firebase's own shape for an Android app id.
var androidAppIDRe = regexp.MustCompile(`^1:\d+:android:[0-9a-f]+$`)

// AppBuilds lists this tenant's builds, newest first.
func (h *Handler) AppBuilds(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	cur, err := h.Store.AppBuilds.Find(r.Context(), bson.M{"tenantId": id},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(20))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows := []models.AppBuild{}
	_ = cur.All(r.Context(), &rows)
	httpx.JSON(w, http.StatusOK, map[string]any{"builds": rows})
}

// DownloadAppBuild hands the file over and then deletes it.
func (h *Handler) DownloadAppBuild(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "buildId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "noto'g'ri id")
		return
	}
	var b models.AppBuild
	if err := h.Store.AppBuilds.FindOne(r.Context(), bson.M{"_id": id}).Decode(&b); err != nil {
		httpx.Error(w, http.StatusNotFound, "build topilmadi")
		return
	}
	if !b.Downloadable() {
		// ⚠️ **Named as what it is.** "Fayl allaqachon yuklab olingan" sends
		// somebody to look on their own machine; a 404 sends them to us.
		httpx.Error(w, http.StatusGone, "fayl allaqachon yuklab olingan")
		return
	}
	f, err := os.Open(b.Path)
	if err != nil {
		// The record says there is a file and the disk disagrees. Recorded on
		// the row rather than only logged: the console has to stop offering it.
		h.finishAppBuild(b.ID, bson.M{
			"status": models.AppFailed, "path": "",
			"error": "fayl serverda topilmadi",
		})
		httpx.Error(w, http.StatusGone, "fayl serverda topilmadi")
		return
	}
	defer f.Close()

	name := fmt.Sprintf("%s-%s.%s", b.Slug, b.VersionName, b.Format)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	if b.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", b.Size))
	}
	sent, err := io.Copy(w, f)
	// ⚠️ **The file is removed only when the whole of it went out.** Deleting on
	// the first byte means a dropped connection costs another nine-minute build,
	// and the retry finds nothing — which reads as the console losing the file.
	if err != nil || (b.Size > 0 && sent != b.Size) {
		return
	}
	_ = f.Close()
	_ = os.Remove(b.Path)
	now := time.Now()
	h.finishAppBuild(b.ID, bson.M{
		"status": models.AppTaken, "path": "",
		"downloadedAt": now, "downloadedBy": h.pressedBy(r),
	})
}

// ---- The runner ----

// StartAppBuilder starts the single worker that drains the queue.
//
// ⚠️ **One worker, not a pool.** The whole reason this is a queue is that the
// machine cannot afford two at once; a pool with a size of one is a pool
// somebody will helpfully raise.
func (h *Handler) StartAppBuilder(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case id := <-h.appBuilds:
				h.runAppBuild(ctx, id)
			}
		}
	}()
}

// runAppBuild does one build, start to finish.
func (h *Handler) runAppBuild(ctx context.Context, id primitive.ObjectID) {
	var b models.AppBuild
	if err := h.Store.AppBuilds.FindOne(ctx, bson.M{"_id": id}).Decode(&b); err != nil {
		return
	}
	now := time.Now()
	h.finishAppBuild(id, bson.M{"status": models.AppBuilding, "startedAt": now})

	// ⚠️ **The version counts up from what this tenant has already had.** Play
	// refuses an upload whose `versionCode` is not higher than the last one, and
	// the refusal arrives at the end of an upload somebody has waited for. Read
	// from our own record rather than from the store, because the store is not
	// something this machine can ask.
	code := h.nextVersionCode(ctx, b.TenantID)
	version := fmt.Sprintf("1.0.%d", code)

	out, err := h.Docker.RunOnce(ctx, provision.RunSpec{
		Image: h.Cfg.AppBuildImage,
		Cmd:   []string{"/opt/keel/deploy/appbuild/build.sh", b.Slug, b.Format},
		Env: map[string]string{
			"KEEL_APP_ROOT":         "/opt/keel",
			"KEEL_APP_HOST":         h.tenantURL(ctx, b.TenantID, b.Slug),
			"KEEL_APP_VERSION_CODE": fmt.Sprintf("%d", code),
			"KEEL_APP_VERSION_NAME": version,
			"KEEL_MAPS_KEY":         h.Cfg.AppMapsKey,
			"KEEL_FB_PROJECT_ID":    h.Cfg.AppFirebaseProject,
			"KEEL_FB_API_KEY":       h.Cfg.AppFirebaseAPIKey,
			"KEEL_FB_SENDER_ID":     h.Cfg.AppFirebaseSender,
			// ⚠️ **Per restaurant, unlike the three above it.** An FCM token is
			// bound to a Firebase app id and the SDK sends the package name with
			// it; one id shared across every restaurant's app is not a supported
			// configuration, and its failure is silent on both sides. Empty
			// builds an app without notifications rather than failing.
			"KEEL_FB_APP_ID": h.androidAppID(ctx, b.TenantID),
		},
		Binds: []string{
			h.Cfg.AppBuildRoot + ":/opt/keel",
		},
		Volumes:  []string{h.Cfg.AppBuildCache + ":/root/.gradle"},
		// ⚠️ **3.5 GB, and the figure is not arbitrary.** Gradle is given a 2 GB
		// heap and the Kotlin compiler runs inside it (build.sh); the rest is
		// the JVM itself, R8 and the tooling. At 3 GB the kernel killed the
		// daemon mid-build and Gradle reported it as "daemon disappeared
		// unexpectedly" — a message that names neither memory nor this limit.
		//
		// ⚠️ Still a ceiling, because this runs on the machine that serves every
		// customer: a build that swaps takes the restaurants with it, and "the
		// site was slow this afternoon" costs far more than a build that died.
		MemoryMB: 3584,
		Timeout:  appBuildTimeout,
	})
	done := time.Now()
	if err != nil || out.ExitCode != 0 {
		msg := out.Output
		if err != nil {
			msg = err.Error() + "\n" + msg
		}
		h.finishAppBuild(id, bson.M{
			"status": models.AppFailed, "finishedAt": done,
			"error": tailLines(msg, 40),
		})
		return
	}

	// ⚠️ **Read off a marked line, never off the last one.** The path still
	// comes from the script rather than being rebuilt here from the slug and a
	// clock — two implementations of "what is this file called" drift on the
	// first change to either — but *which* line carries it has to be something
	// the output cannot push around. Docker interleaves stdout and stderr by
	// write time and the Android tooling writes progress to stderr, so the
	// last line is whatever happened to be written last: this shipped reading
	// `Preparing "Install Android SDK Build-Tools 35…"` as a filename.
	path := markedValue(out.Output, "KEEL_ARTIFACT=")
	info, statErr := os.Stat(path)
	if statErr != nil {
		// ⚠️ **The output is kept.** The first version of this replaced it with
		// a one-line message, which is exactly the case where somebody needs to
		// see what the build actually said — and the only way to find out was
		// to run the whole thing again by hand.
		h.finishAppBuild(id, bson.M{
			"status": models.AppFailed, "finishedAt": done,
			"error": "build tugadi, lekin fayl topilmadi (" + path + ")\n\n" +
				tailLines(out.Output, 40),
		})
		return
	}
	h.finishAppBuild(id, bson.M{
		"status": models.AppReady, "finishedAt": done,
		"path": path, "size": info.Size(), "sha256": fileHash(path),
		"applicationId": "uz.keel.app." + slugID(b.Slug),
		"versionCode":   code, "versionName": version,
	})
}

// androidAppID is this restaurant's Firebase app id, or empty.
func (h *Handler) androidAppID(ctx context.Context, id primitive.ObjectID) string {
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(ctx, bson.M{"_id": id}).Decode(&t); err != nil {
		return ""
	}
	return t.AndroidAppID
}

// nextVersionCode is one past the highest this tenant has ever built.
//
// ⚠️ **Never a timestamp and never a count of rows.** Play refuses an upload
// whose code is not strictly higher than the last accepted one, so it has to
// rise monotonically — and a count would repeat itself the first time a row is
// deleted.
func (h *Handler) nextVersionCode(ctx context.Context, tenant primitive.ObjectID) int {
	var last models.AppBuild
	err := h.Store.AppBuilds.FindOne(ctx,
		bson.M{"tenantId": tenant, "versionCode": bson.M{"$gt": 0}},
		options.FindOne().SetSort(bson.D{{Key: "versionCode", Value: -1}}),
	).Decode(&last)
	if err != nil {
		return 1
	}
	return last.VersionCode + 1
}

// tenantURL is the address the build reads the restaurant's profile from.
func (h *Handler) tenantURL(ctx context.Context, id primitive.ObjectID, slug string) string {
	var t models.Tenant
	if err := h.Store.Tenants.FindOne(ctx, bson.M{"_id": id}).Decode(&t); err == nil {
		// ⚠️ **The customer's own domain when they have one.** The app's server
		// address is baked into the build and never asked again; pointing it at
		// `slug.keel.uz` for a restaurant that has moved to their own domain
		// would work today and break the afternoon somebody tidies up DNS.
		if len(t.Domains) > 0 && t.Domains[0] != "" {
			return "https://" + t.Domains[0]
		}
	}
	return "https://" + slug + ".keel.uz"
}

// finishAppBuild writes one update, ignoring the error.
//
// ⚠️ **Its own short context.** The build's context is cancelled the moment it
// times out, and an update that inherited it would never be written — leaving a
// row saying "building" for a build that died half an hour ago.
func (h *Handler) finishAppBuild(id primitive.ObjectID, set bson.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, _ = h.Store.AppBuilds.UpdateByID(ctx, id, bson.M{"$set": set})
}

// slugID is the slug as it appears in an application id: letters and digits
// only, matching what build.sh derives. ⚠️ Two implementations of one rule, and
// this one exists only so the console can *display* the id — the build's own is
// what actually ships.
func slugID(slug string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(slug) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// markedValue is what the script put behind a marker, or empty.
//
// ⚠️ **The last match, not the first.** A retried build inside one container
// would print two, and the one that matters is the one that produced the file
// this run is about to hand over.
func markedValue(s, prefix string) string {
	out := ""
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			out = strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return out
}

func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// fileHash is the artifact's SHA-256.
//
// ⚠️ **Computed while the file exists and kept after it is gone.** It is the
// only way to answer "is the APK on my laptop the one you built me" once the
// artifact has been downloaded and deleted.
func fileHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return ""
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// pressedBy is whose name goes on the row.
//
// ⚠️ **Recorded rather than derived later.** "Who asked for this build" and "who
// took the file" are the two questions a signing key makes worth asking, and a
// token that has since expired cannot answer either.
func (h *Handler) pressedBy(r *http.Request) string {
	if u, err := h.actor(r); err == nil && u != nil {
		return u.Name
	}
	return ""
}
