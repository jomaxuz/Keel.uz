package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Who works in the console, and what each of them may see.
//
// Until now there was one account: the platform's owner. That works while selling is
// one person's job and stops working the moment somebody is hired to do it — an agent
// with the owner's login can suspend a paying customer, read every restaurant's
// numbers and change the billing.
//
// Four roles, and the boundaries are drawn where the damage is:
//
//   - **owner** — everything. The only role that can create accounts, read the
//     activity log and touch the server itself.
//   - **admin** — runs the platform day to day: provisioning, domains, invoices.
//     Cannot create accounts and cannot read the log, because an account that can
//     grant itself more is not a boundary, and a log its subject can edit is not a
//     record.
//   - **manager** — sales oversight: every customer and who brought them in, no
//     server controls and no billing changes.
//   - **agent** — their own customers only, and ⚠️ **no restaurant statistics at
//     all**. An agent needs to know who they signed up and what they promised; a
//     restaurant's turnover is that restaurant's business, and handing it to a
//     salesperson is a leak with a friendly name.
const (
	RoleOwner   = "owner"
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleAgent   = "agent"
)

// StaffRoles is the set an account may hold, in order of reach.
var StaffRoles = []string{RoleOwner, RoleAdmin, RoleManager, RoleAgent}

// CanSeeAllTenants reports whether this role reads the whole customer list.
//
// ⚠️ A single function rather than a check per endpoint. The rule "an agent sees only
// their own" has to hold on the list, the card, the live figures, the invoices and
// every future screen — and a rule spelled out five times is a rule that will be
// spelled out wrong the sixth.
func CanSeeAllTenants(role string) bool {
	return role == RoleOwner || role == RoleAdmin || role == RoleManager
}

// CanSeeStats reports whether this role may read a customer's own business figures.
func CanSeeStats(role string) bool {
	return role == RoleOwner || role == RoleAdmin
}

// CanManageStaff reports whether this role may create accounts. Owner only.
func CanManageStaff(role string) bool { return role == RoleOwner }

// CanSeeLog reports whether this role may read the activity log. Owner only — the
// point of the log is answering "who did this", and a record its subjects can read
// selectively is a record they can argue with.
func CanSeeLog(role string) bool { return role == RoleOwner }

// CanProvision reports whether this role may create, suspend or delete a customer.
func CanProvision(role string) bool {
	return role == RoleOwner || role == RoleAdmin
}

// CanBill reports whether this role may issue or void invoices.
func CanBill(role string) bool { return role == RoleOwner || role == RoleAdmin }

// Visit is a planned or completed call on a business.
//
// ⚠️ **Planned before it happens, judged after.** An agent's day is a list of places
// to walk into, and the useful record is not "I visited twelve" but "this one said
// come back in a month". So a visit is written in advance with a date, and closed with
// an outcome and a sentence — and the sentence is required on a negative one, because
// "no" without a reason teaches nobody anything.
type Visit struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	// Who is going. Set from the session, never from the request: a visit log an agent
	// can attribute to somebody else is not a log.
	AgentID   primitive.ObjectID `bson:"agentId" json:"agentId"`
	AgentName string             `bson:"agentName" json:"agentName"`

	// The place. Free text, because most of these are not customers yet — that is
	// the entire point of visiting them.
	Place   string `bson:"place" json:"place"`
	Address string `bson:"address,omitempty" json:"address,omitempty"`
	Phone   string `bson:"phone,omitempty" json:"phone,omitempty"`
	// Set once they become a customer, so the visit that produced a signup can be
	// traced to it.
	TenantID primitive.ObjectID `bson:"tenantId,omitempty" json:"tenantId,omitempty"`

	// When they mean to go. Kept as the day rather than a timestamp: an agent plans
	// "Thursday", not "Thursday at 14:20".
	PlannedFor string `bson:"plannedFor" json:"plannedFor"`

	// planned | done
	Status string `bson:"status" json:"status"`
	// positive | negative | callback — only once the visit is done.
	Outcome   string     `bson:"outcome,omitempty" json:"outcome,omitempty"`
	Comment   string     `bson:"comment,omitempty" json:"comment,omitempty"`
	VisitedAt *time.Time `bson:"visitedAt,omitempty" json:"visitedAt,omitempty"`
	// A promise to come back. Same reasoning as the call centre's callbacks: a
	// commitment with no date is not a commitment.
	NextAt string `bson:"nextAt,omitempty" json:"nextAt,omitempty"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

// Visit statuses and outcomes. Stored, so never renamed.
const (
	VisitPlanned = "planned"
	VisitDone    = "done"

	OutcomePositive = "positive"
	OutcomeNegative = "negative"
	OutcomeCallback = "callback"
)

// ConsoleLog is one thing somebody did in the console.
//
// ⚠️ Owner-only, and it exists for one question: who signed this customer up, and who
// changed what afterwards. Attribution that lives only in `tenant.createdBy` answers
// the first and loses the second the moment a name is edited.
type ConsoleLog struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ActorID   primitive.ObjectID `bson:"actorId,omitempty" json:"actorId,omitempty"`
	Actor     string             `bson:"actor" json:"actor"`
	ActorRole string             `bson:"actorRole,omitempty" json:"actorRole,omitempty"`
	// A stable id like "tenant.create" — translated in the console, never shown raw.
	Action string    `bson:"action" json:"action"`
	Target string    `bson:"target,omitempty" json:"target,omitempty"`
	Detail string    `bson:"detail,omitempty" json:"detail,omitempty"`
	At     time.Time `bson:"at" json:"at"`
}
