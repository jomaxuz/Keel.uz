package uz.keel.courier

import uz.keel.app.KeelLook
import uz.keel.app.deviceInfoFor
import uz.keel.courier.data.DeviceInfo
import uz.keel.courier.data.KeelApi
import uz.keel.design.TokenStore

// What the courier's half of Keel owns — made the first time it is opened, or
// the first time the shift's location service needs to send.
class CourierGraph(val tokens: TokenStore, look: KeelLook) {

    val api: KeelApi = KeelApi(tokens)
    val prefs: Prefs = Prefs(tokens, look)

    init {
        // ⚠️ `keel-courier`, not `courier` — see `deviceInfoFor`.
        api.device = deviceInfoFor("keel-courier", tokens).let { DeviceInfo(it.id, it.app, it.name) }
        tokens.read(TokenStore.SERVER_ADDRESS)?.let { api.useServer(it) }
    }
}
