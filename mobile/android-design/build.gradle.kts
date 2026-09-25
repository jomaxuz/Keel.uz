// The design system and the plumbing both phone applications share.
//
// ⚠️ **One copy, and the reason is the whole history of this repository.** The
// waiter app and the owner app are the same product on two people's phones: the
// same glass, the same orange, the same three languages, the same server
// address rule. Written twice they drift — and the drift lands on a phone in a
// restaurant, months later, as "the owner's app looks different now". The dark
// theme alone was corrected twice in one afternoon; doing that twice again, in
// two places, is the failure this module exists to prevent.
//
// ⚠️ **A module included by path, not a published library.** Both apps are
// separate Gradle builds and neither moves; each `settings.gradle.kts` points
// here. A published artifact would mean a version number and a release step
// between changing a colour and seeing it.
plugins {
    alias(libs.plugins.android.library)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "uz.keel.design"
    compileSdk = 36
    defaultConfig { minSdk = 26 }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlin {
        compilerOptions {
            jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
            freeCompilerArgs.add("-opt-in=androidx.compose.material3.ExperimentalMaterial3Api")
        }
    }
    buildFeatures { compose = true }
}

dependencies {
    api(platform(libs.compose.bom))
    api(libs.compose.ui)
    api(libs.compose.ui.graphics)
    api(libs.compose.material3)
    api(libs.compose.material.icons)
    api(libs.compose.ui.tooling.preview)
    api(libs.coroutines.android)
    api(libs.serialization.json)
    api(libs.ktor.client.okhttp)
    api(libs.ktor.client.content.negotiation)
    api(libs.ktor.serialization.json)
    api(libs.security.crypto)
    api(libs.core.ktx)
    // The location gate (LocationGate.kt): one copy of the permission, the
    // "location is off" dialog and the fix, for every app that punches or tracks.
    implementation(libs.activity.compose)
    implementation(libs.play.services.location)
    implementation(libs.coroutines.play.services)
    debugImplementation(libs.compose.ui.tooling)
}
