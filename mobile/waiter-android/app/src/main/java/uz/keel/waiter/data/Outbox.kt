package uz.keel.waiter.data

import android.content.Context
import androidx.room.ColumnInfo
import androidx.room.Dao
import androidx.room.Database
import androidx.room.Entity
import androidx.room.Insert
import androidx.room.PrimaryKey
import androidx.room.Query
import androidx.room.Room
import androidx.room.RoomDatabase
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.json.Json
import java.util.UUID

// The phone's own disk, for the minutes the server is not there.
//
// ⚠️ **The failure this exists for is not a rare one.** A restaurant's wifi
// drops for thirty seconds several times a week. Until now every tap made during
// those seconds came back as "qo'shib bo'lmadi" and was simply lost — the waiter
// retyped an order they had already taken, at a table, in front of the guest.
//
// ⚠️ **This is an outbox, not the till's offline mode, and the difference is
// deliberate.** The Windows till can open a whole check with no server, mint its
// own number and reconcile later (`lib/offline/checks.ts`) — it has to, it is
// the machine that takes the money. A waiter's phone cannot open a check
// offline anyway: the check is created by the server, and the table it belongs
// to is shared with a cashier and a kitchen screen. What the phone *can* do is
// never lose an edit to a check that already exists, and that is what this is.
//
// ⚠️ **Every operation here is safe to repeat except one.** `qty = 3`,
// `served = true`, "void this line id", "fire what is unfired" all state an
// absolute: applying them twice lands on the same answer. Adding dishes says
// "one more", so it — and only it — carries an `opId` the server remembers
// (`Order.AppliedOps`). That asymmetry is the reason this queue can retry at all.

@Entity(tableName = "outbox")
data class OutboxOp(
    @PrimaryKey val id: String,
    /** Which check this belongs to. ⚠️ Kept so a screen can say what is still
     *  unsent for the table somebody is standing at, rather than only a total. */
    @ColumnInfo(name = "check_id") val checkId: String,
    val kind: String,
    /** The operation's arguments, as JSON. ⚠️ One column rather than a table per
     *  verb: the queue's job is to hand bytes back unchanged, and a schema that
     *  understood them would need a migration every time an endpoint grows a
     *  field. */
    val payload: String,
    @ColumnInfo(name = "created_at") val createdAt: Long,
    /** How many times sending has been attempted. */
    val attempts: Int = 0,
)

@Dao
interface OutboxDao {
    // ⚠️ Ordered by when it was made, and that ordering is the correctness of
    // the whole queue: "add two coffees" then "make it three" is a different
    // evening from the reverse.
    @Query("SELECT * FROM outbox ORDER BY created_at ASC")
    suspend fun all(): List<OutboxOp>

    @Query("SELECT COUNT(*) FROM outbox")
    suspend fun count(): Int

    @Insert
    suspend fun add(op: OutboxOp)

    @Query("DELETE FROM outbox WHERE id = :id")
    suspend fun remove(id: String)

    @Query("UPDATE outbox SET attempts = attempts + 1 WHERE id = :id")
    suspend fun failed(id: String)

    @Query("DELETE FROM outbox WHERE attempts >= :limit")
    suspend fun dropExhausted(limit: Int)
}

@Database(entities = [OutboxOp::class], version = 1, exportSchema = false)
abstract class OutboxDb : RoomDatabase() {
    abstract fun ops(): OutboxDao
}

/** What a queued operation is.
 *
 *  ⚠️ **Named rather than a lambda.** A queue holds work across a process death
 *  — the phone is killed by the operating system without warning, which is this
 *  app's version of a power cut — and a closure cannot be written to a disk. */
enum class OpKind { AddLines, LineQty, VoidLine, CommentLine, LineServed, Fire }

class Outbox(
    context: Context,
    private val api: KeelApi,
) {
    private val db = Room.databaseBuilder(
        context.applicationContext, OutboxDb::class.java, "keel-waiter-outbox",
    ).build()

    private val dao = db.ops()
    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    /** ⚠️ One flush at a time. Two would send the same operation twice and
     *  reorder the rest — and the reordering is the half nobody would notice. */
    private val flushing = Mutex()

    private val _pending = MutableStateFlow(0)

    /** How much is waiting. ⚠️ Shown to the waiter rather than kept quiet: an
     *  app that is holding four dishes it has not managed to send looks exactly
     *  like an app that sent them, and the difference reaches the guest. */
    val pending: StateFlow<Int> = _pending.asStateFlow()

    init {
        scope.launch {
            // ⚠️ Anything left from a previous life goes out before anything new
            // is taken: the queue survives being killed, which is the only
            // reason it is on a disk rather than in memory.
            dao.dropExhausted(MAX_ATTEMPTS)
            refreshCount()
            flush()
        }
    }

    /** How many times an operation is retried before it is given up on.
     *
     *  ⚠️ **A bounded number, and giving up is the honest end.** An operation the
     *  server refuses on its merits — a dish that is now sold out, a line the
     *  cashier already voided — will be refused every time; retried forever it
     *  becomes a queue that never drains and blocks everything behind it. The
     *  waiter's own next action is what corrects those, and they are standing at
     *  the table. */
    private val MAX_ATTEMPTS = 8

    private suspend fun refreshCount() {
        _pending.value = dao.count()
    }

    /** What happened to an operation.
     *
     *  ⚠️ **Three outcomes, not a `Result`, and the third is the one a screen
     *  gets wrong.** "Sent" and "the server refused it" are the two a boolean
     *  can carry; "held on the phone and it *will* happen" is neither, and
     *  drawing it as a failure is what makes a waiter re-tap a dish that is
     *  already queued. The same distinction the launch screen makes between a
     *  refusal and a network, in the one place it costs a guest money.
     */
    sealed interface Outcome<out T> {
        data class Sent<T>(val value: T) : Outcome<T>
        data class Refused(val error: ApiError) : Outcome<Nothing>
        data object Queued : Outcome<Nothing>
    }

    /** Try now, and queue if the network is not there.
     *
     *  ⚠️ **The distinction is `ApiError` versus anything else, exactly as the
     *  launch screen makes it.** A server that answered — a sold-out dish, a
     *  refused write-off, an expired token — has said something the waiter needs
     *  to read now; queueing that would hide a real answer behind a spinner and
     *  deliver it as a surprise ten minutes later. Only a request that never
     *  arrived is worth holding on to. */
    suspend fun <T> send(
        checkId: String,
        kind: OpKind,
        payload: String,
        call: suspend (opId: String) -> T,
    ): Outcome<T> {
        val opId = UUID.randomUUID().toString()
        // ⚠️ **Behind the queue, not past it, while anything is waiting.** Sent
        // straight away, "make it three" could reach the server before the "add
        // two" still sitting on the disk — the ordering the whole queue exists to
        // keep. And while the network is down every direct attempt costs a full
        // timeout before it is queued anyway, which is the wait people felt as a
        // frozen menu. Queued here, it is drained in order the moment the flush
        // gets through.
        if (_pending.value > 0) {
            enqueue(opId, checkId, kind, payload)
            flushSoon()
            return Outcome.Queued
        }
        return try {
            Outcome.Sent(call(opId))
        } catch (e: ApiError) {
            Outcome.Refused(e)
        } catch (e: CancellationException) {
            // ⚠️ The caller went away (the screen closed); that is not a network
            // failure, and writing it to the queue from a cancelled coroutine
            // fails anyway. The callers that must finish run it NonCancellable.
            throw e
        } catch (e: Throwable) {
            // ⚠️ Written with the id the attempt already used. A resend that
            // minted a fresh one would be a new tap to the server, and the
            // dedupe it relies on would never match.
            enqueue(opId, checkId, kind, payload)
            Outcome.Queued
        }
    }

    private suspend fun enqueue(opId: String, checkId: String, kind: OpKind, payload: String) {
        dao.add(
            OutboxOp(
                id = opId, checkId = checkId, kind = kind.name,
                payload = payload, createdAt = System.currentTimeMillis(),
            ),
        )
        refreshCount()
    }

    /** Send everything that is waiting, oldest first. */
    fun flushSoon() {
        scope.launch { flush() }
    }

    suspend fun flush() {
        if (!flushing.tryLock()) return
        try {
            // ⚠️ **Until it is empty or stuck, not one pass.** A tap queued while
            // this pass was running used to wait out the whole retry interval
            // with the network already back.
            var stuck = false
            while (!stuck) {
                val ops = dao.all()
                if (ops.isEmpty()) break
                for (op in ops) {
                    try {
                        replay(op)
                        dao.remove(op.id)
                    } catch (e: ApiError) {
                        // ⚠️ **The server answered, so this will not improve by
                        // being asked again.** A voided line, a sold-out dish, a
                        // closed check: dropped rather than retried, or one dead
                        // operation holds up every live one behind it.
                        dao.remove(op.id)
                    } catch (e: CancellationException) {
                        throw e
                    } catch (e: Throwable) {
                        // Still no network. Left in place, counted, and the rest
                        // of the queue is not attempted — ⚠️ order is the
                        // correctness here, and skipping ahead would apply "make
                        // it three" before "add two".
                        dao.failed(op.id)
                        dao.dropExhausted(MAX_ATTEMPTS)
                        stuck = true
                        break
                    }
                }
                refreshCount()
            }
            refreshCount()
        } finally {
            flushing.unlock()
        }
    }

    /** ⚠️ Retried on a schedule as well as on demand: the ordinary way out is
     *  the network coming back by itself — walking out of the cellar, the router
     *  rebooting — and nothing tells the app when that happened. */
    fun startRetrying() {
        scope.launch {
            while (true) {
                delay(RETRY_MS)
                if (_pending.value > 0) flush()
            }
        }
    }

    private suspend fun replay(op: OutboxOp) {
        when (OpKind.valueOf(op.kind)) {
            OpKind.AddLines -> {
                val items = json.decodeFromString<List<AddLineRequest>>(op.payload)
                // ⚠️ The id the first attempt used, not a new one. This is the
                // one operation a repeat would charge a guest for, and the
                // server refuses a repeat by exactly this id.
                api.addLines(op.checkId, items, op.id)
            }
            OpKind.LineQty -> {
                val a = json.decodeFromString<LineQtyPayload>(op.payload)
                api.lineQty(op.checkId, a.lineId, a.qty)
            }
            OpKind.VoidLine -> {
                val a = json.decodeFromString<VoidPayload>(op.payload)
                api.voidLine(op.checkId, a.lineId, a.reason, a.pin)
            }
            OpKind.CommentLine -> {
                val a = json.decodeFromString<CommentPayload>(op.payload)
                api.commentLine(op.checkId, a.lineId, a.comment)
            }
            OpKind.LineServed -> {
                val a = json.decodeFromString<ServedPayload>(op.payload)
                api.lineServed(op.checkId, a.lineId, a.served)
            }
            OpKind.Fire -> api.fire(op.checkId)
        }
    }

    companion object {
        /** Long enough to be nothing on a battery, short enough that a phone
         *  walking back into signal drains while it is still in the hand. */
        const val RETRY_MS = 8_000L
    }
}

// The argument shapes, so a queued operation can be written to a disk and read
// back by a different process than the one that made it.
@kotlinx.serialization.Serializable
data class LineQtyPayload(val lineId: String, val qty: Int)

@kotlinx.serialization.Serializable
data class VoidPayload(val lineId: String, val reason: String = "", val pin: String = "")

@kotlinx.serialization.Serializable
data class CommentPayload(val lineId: String, val comment: String)

@kotlinx.serialization.Serializable
data class ServedPayload(val lineId: String, val served: Boolean)
