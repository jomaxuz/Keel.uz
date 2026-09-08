# kotlinx.serialization keeps its generated serializers on the class it
# describes; R8 cannot see the reference and strips them, and the failure is a
# runtime "Serializer for class ... not found" on the first API call — which is
# the menu, i.e. the whole app.
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.**
-keepclassmembers class uz.keel.guest.data.** {
    *** Companion;
}
-keepclasseswithmembers class uz.keel.guest.data.** {
    kotlinx.serialization.KSerializer serializer(...);
}
