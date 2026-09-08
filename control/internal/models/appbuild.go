package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- One build of one restaurant's Android app ----
//
// ⚠️ **The record outlives the file, and that is the whole shape of this
// document.** The artifact is deleted the moment somebody has downloaded it —
// it is two and a half megabytes per build on a machine that also serves every
// customer, and a directory of last year's APKs is a directory nobody prunes.
// What must not be deleted is the *fact*: which version was built, when, by
// whom, from which commit, and who took it away. "Which build is on the store"
// is a question asked months later, and a filesystem cannot answer it.
//
// ⚠️ **A queue rather than a background goroutine per press.** Gradle and the
// Kotlin compiler together want more memory than that machine has spare beside
// the restaurants; two at once is a box that swaps, and "the site was slow this
// afternoon" is a far more expensive failure than a build somebody waited for.
type AppBuild struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID primitive.ObjectID `bson:"tenantId" json:"tenantId"`
	// ⚠️ Frozen beside the id, because the tenant can be renamed and this is
	// what the artifact was actually named after.
	Slug string `bson:"slug" json:"slug"`

	// "apk" or "aab". ⚠️ Asked rather than assumed: an APK is what somebody
	// installs on a phone this afternoon and an AAB is what Play accepts, and
	// handing over the wrong one wastes a nine-minute build.
	Format string `bson:"format" json:"format"`

	// AppQueued → AppBuilding → AppReady → AppTaken, or AppFailed.
	Status string `bson:"status" json:"status"`

	// What the build produced. ⚠️ Kept after the file is gone, on purpose: an
	// owner asking "which version did I upload?" is asking about a build whose
	// artifact was deleted the day it was downloaded.
	ApplicationID string `bson:"applicationId,omitempty" json:"applicationId,omitempty"`
	VersionCode   int    `bson:"versionCode,omitempty" json:"versionCode,omitempty"`
	VersionName   string `bson:"versionName,omitempty" json:"versionName,omitempty"`

	// Where the artifact is, while it still exists.
	//
	// ⚠️ **Cleared when the file is deleted rather than left dangling.** A path
	// pointing at nothing is a download button that produces a 500, and the
	// person pressing it has no way to know the file was theirs to begin with.
	Path string `bson:"path,omitempty" json:"-"`
	Size int64  `bson:"size,omitempty" json:"size,omitempty"`
	// ⚠️ **Kept forever, unlike the file.** It is the only way to answer "is the
	// APK on my laptop the one you built me" after the artifact is gone.
	SHA256 string `bson:"sha256,omitempty" json:"sha256,omitempty"`

	// The tail of the build's output, kept only when it failed: a successful
	// Gradle run is two thousand lines nobody will ever read.
	Error string `bson:"error,omitempty" json:"error,omitempty"`

	// Who pressed the button, and who took the file away. ⚠️ Two names, because
	// they are two different acts and often two different people.
	By           string     `bson:"by,omitempty" json:"by,omitempty"`
	DownloadedBy string     `bson:"downloadedBy,omitempty" json:"downloadedBy,omitempty"`
	DownloadedAt *time.Time `bson:"downloadedAt,omitempty" json:"downloadedAt,omitempty"`

	CreatedAt  time.Time  `bson:"createdAt" json:"createdAt"`
	StartedAt  *time.Time `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	FinishedAt *time.Time `bson:"finishedAt,omitempty" json:"finishedAt,omitempty"`
}

const (
	// AppQueued is written and waiting for the machine.
	AppQueued = "queued"
	// AppBuilding is in the container right now.
	AppBuilding = "building"
	// AppReady has a file somebody can download.
	AppReady = "ready"
	// AppTaken has been downloaded, and the file is gone.
	//
	// ⚠️ **Its own state rather than a flag on "ready".** "There is a file" and
	// "there was a file" are what the console draws a button from, and a
	// boolean beside a status is two facts that can disagree — which shows up
	// as a download button that answers 404.
	AppTaken = "taken"
	// AppFailed carries the tail of the output.
	AppFailed = "failed"
)

// Downloadable reports whether there is still a file behind this record.
func (b AppBuild) Downloadable() bool { return b.Status == AppReady && b.Path != "" }
