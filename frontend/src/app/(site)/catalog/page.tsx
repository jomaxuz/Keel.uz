// The catalogue, at the address a shop's guest should see.
//
// ⚠️ **The same page as `/menu`, not a copy of it.** Two routes render one
// implementation and each sends the guest on when the address is not this
// business's — a restaurant asking for `/catalog` lands on `/menu` and a shop
// asking for `/menu` lands here. A second copy of the page would be the copy
// that drifts, and it would drift in the half nobody is looking at.
export { default, generateMetadata } from "../menu/page";
