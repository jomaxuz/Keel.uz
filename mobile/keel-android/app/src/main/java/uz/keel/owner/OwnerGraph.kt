package uz.keel.owner

import uz.keel.app.KeelLook
import uz.keel.app.deviceInfoFor
import uz.keel.design.TokenStore
import uz.keel.owner.data.DeviceInfo
import uz.keel.owner.data.KeelApi

// What the owner's half of Keel owns — made the first time it is opened.
class OwnerGraph(val tokens: TokenStore, look: KeelLook) {

    val api: KeelApi = KeelApi(tokens)
    val prefs: Prefs = Prefs(tokens, look)

    init {
        // ⚠️ `keel-owner`, not `owner` — Keel Owner may still be installed beside
        // this one, and a shared key makes each evict the other. See
        // `deviceInfoFor`.
        api.device = deviceInfoFor("keel-owner", tokens).let { DeviceInfo(it.id, it.app, it.name) }
        tokens.read(TokenStore.SERVER_ADDRESS)?.let { api.useServer(it) }
    }
}
