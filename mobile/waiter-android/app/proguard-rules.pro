# What R8 must not remove.
#
# ⚠️ **Every rule here is for something reached by reflection, and every one of
# them fails silently.** Shrinking removes what nothing calls — and nothing
# *calls* a generated serializer, a Room implementation or a Ktor engine; they
# are looked up by name at runtime. The result is not a build error: it is a
# release that installs, opens, and then cannot parse a check. Debug builds have
# minification off, so none of it is visible until the build somebody ships.

# ---- kotlinx.serialization ----
#
# ⚠️ The compiler plugin generates a `Companion.serializer()` and a `$$serializer`
# class per @Serializable type, and they are found through the companion at
# runtime. Removed, every API response fails to decode — which on this app is
# every screen at once.
-keepattributes *Annotation*, InnerClasses, Signature, RuntimeVisible*Annotations
-dontnote kotlinx.serialization.**

-if @kotlinx.serialization.Serializable class **
-keepclassmembers class <1> {
    static <1>$Companion Companion;
}
-if @kotlinx.serialization.Serializable class ** {
    static **$* *;
}
-keepclassmembers class <2>$<3> {
    kotlinx.serialization.KSerializer serializer(...);
}
-keep,includedescriptorclasses class uz.keel.waiter.**$$serializer { *; }
-keepclassmembers class uz.keel.waiter.** {
    *** Companion;
}
-keepclasseswithmembers class uz.keel.waiter.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# ---- Ktor ----
#
# ⚠️ The engine is picked up through a service loader, so nothing references
# OkHttp's implementation by name. Without this the client throws "no engine"
# on the first request — the sign-in.
-keep class io.ktor.client.engine.okhttp.** { *; }
-keep class io.ktor.** { *; }
-dontwarn io.ktor.**
-dontwarn org.slf4j.**

# ---- OkHttp / Okio ----
-dontwarn okhttp3.**
-dontwarn okio.**
-keepnames class okhttp3.internal.publicsuffix.PublicSuffixDatabase

# ---- Room ----
#
# ⚠️ The generated `_Impl` is instantiated by name from the database builder.
# Removed, the outbox cannot open — and the outbox is what holds a waiter's
# taps through an outage, so it fails on exactly the evening it matters.
-keep class uz.keel.waiter.data.OutboxDb_Impl { *; }
-keep class * extends androidx.room.RoomDatabase { <init>(); }
-dontwarn androidx.room.paging.**

# ---- Firebase messaging ----
#
# The service is named in the manifest and constructed by the framework.
-keep class uz.keel.waiter.push.KitchenMessagingService { *; }

# ---- Coil ----
-dontwarn coil3.**

# ⚠️ Kept so a crash report from a restaurant names a line rather than "a.b.c".
# The mapping file is what turns it back (app/build/outputs/mapping/release).
-keepattributes SourceFile,LineNumberTable
-renamesourcefileattribute SourceFile
