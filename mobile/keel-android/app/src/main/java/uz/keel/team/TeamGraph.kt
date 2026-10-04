package uz.keel.team

import uz.keel.app.KeelLook
import uz.keel.app.deviceInfoFor
import uz.keel.design.TokenStore
import uz.keel.team.data.DeviceInfo
import uz.keel.team.data.KeelApi

// What the team's half of Keel owns — made the first time it is opened.
class TeamGraph(val tokens: TokenStore, look: KeelLook) {

    val api: KeelApi = KeelApi(tokens)
    val prefs: Prefs = Prefs(tokens, look)

    /** A count in progress. ⚠️ Owned by the process rather than by the screen,
     *  because half an hour of walking a store must survive a tab being tapped
     *  — and here, a trip to the floor and back — see CountDraft. */
    val counting = CountDraft()

    init {
        // ⚠️ **`keel-staff`, the same key as the floor.** One staff account, one
        // phone: the waiter's section and this one are the same person, and two
        // keys would let one account hold two phones through one app.
        api.device = deviceInfoFor("keel-staff", tokens).let { DeviceInfo(it.id, it.app, it.name) }
        tokens.read(TokenStore.SERVER_ADDRESS)?.let { api.useServer(it) }
    }
}
