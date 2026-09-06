# What must survive R8.
#
# ⚠️ **The file exists even when it is nearly empty**, because `proguardFiles`
# names it: a missing file fails the release build and nothing else, which is a
# failure discovered at the worst moment — the first time somebody tries to
# produce the build that goes on a wall.
#
# kotlinx.serialization and Ktor ship their own consumer rules; media3 does too.
# What is listed here is only what this application adds.

# ⚠️ The wire models are constructed by name, from JSON, and never by us. R8
# cannot see the call, so without this a released television parses an empty
# playlist and plays nothing — silently, which is this app's whole failure mode.
-keep,includedescriptorclasses class uz.keel.team.data.**$$serializer { *; }
-keepclassmembers class uz.keel.team.data.** {
    *** Companion;
}
-keepclasseswithmembers class uz.keel.team.data.** {
    kotlinx.serialization.KSerializer serializer(...);
}
