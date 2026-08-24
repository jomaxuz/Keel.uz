package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- Which store an ingredient is kept in, in this branch ----
//
// ⚠️ **The two halves of the model belong to different owners, and the shortcut
// broke on the second branch.** An ingredient is a *brand* fact — the menu's
// tech cards name it by id, so a chain's three kitchens cook from one catalogue
// and must share one row. A store is a *branch* fact — it is a physical room
// with a door. `ingredient.warehouseId` tried to be both, which works exactly
// as long as there is one branch.
//
// With two, the failure is silent and total: potatoes filed into Chilonzor's
// kitchen store could not be counted in Yunusobod at all (its sheet skipped
// every ingredient whose store belonged to someone else), and Yunusobod's
// consumption was filed against Chilonzor's shelf. Nothing errored. The stock
// screen simply described the wrong building.
//
// ⚠️ **The original rule survives, one word longer: one ingredient, one
// warehouse — *per branch*.** A restaurant genuinely keeping lemons behind the
// bar *and* in the kitchen of the *same* branch still makes two ingredients,
// for the reasons in warehouse.go. What is now allowed is the thing that was
// always true and could not be said: the same lemon lives in the bar here and
// in the kitchen there.
//
// ⚠️ **No placement is not an error, it is the undivided store** — every
// install that never split its stores, and every ingredient added since. The
// same zero-value rule as an empty `mapProvider` meaning 2GIS.
type IngredientPlacement struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BranchID     primitive.ObjectID `bson:"branchId" json:"branchId"`
	IngredientID primitive.ObjectID `bson:"ingredientId" json:"ingredientId"`
	// ⚠️ The zero id is stored rather than the document being deleted: "this
	// branch keeps it in the undivided store" and "nobody has said" are the
	// same answer today, but an explicit row is what lets the panel show the
	// choice as made rather than as blank.
	WarehouseID primitive.ObjectID `bson:"warehouseId,omitempty" json:"warehouseId,omitempty"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}
