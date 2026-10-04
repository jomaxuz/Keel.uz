# ---- kotlinx.serialization: every wire model in all four roles ----
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
-keep,includedescriptorclasses class uz.keel.**$$serializer { *; }
-keepclassmembers class uz.keel.** {
    *** Companion;
}
-keepclasseswithmembers class uz.keel.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# ---- Ktor on OkHttp ----
-keep class io.ktor.client.engine.okhttp.** { *; }
-keep class io.ktor.** { *; }
-dontwarn io.ktor.**
-dontwarn org.slf4j.**
-dontwarn okhttp3.**
-dontwarn okio.**
-keepnames class okhttp3.internal.publicsuffix.PublicSuffixDatabase

# ---- Room (the waiter's offline outbox) ----
-keep class uz.keel.waiter.data.OutboxDb_Impl { *; }
-keep class * extends androidx.room.RoomDatabase { <init>(); }
-dontwarn androidx.room.paging.**

# ---- Services the manifest names ----
-keep class uz.keel.app.push.KeelMessagingService { *; }
-keep class uz.keel.courier.location.LocationService { *; }

-dontwarn coil3.**
-keepattributes SourceFile,LineNumberTable
-renamesourcefileattribute SourceFile
