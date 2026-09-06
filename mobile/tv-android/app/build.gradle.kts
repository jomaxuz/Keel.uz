// ⚠️ Imported rather than written out in full: inside a Gradle Kotlin DSL
// script `java` resolves to the Java plugin's extension, not to the package, so
// `java.util.Properties` does not compile and the error names `util`.
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
}

// Where this build is signed from.
//
// ⚠️ **Its own key, like the waiter's and the owner's.** A signing key cannot be
// rotated: an application on the store signed by a lost key can never be updated
// again. One key across three applications makes them one blast radius, and this
// one is handed over separately anyway — a television is installed by whoever
// mounts it, months apart from the phones.
//
// ⚠️ **Read from outside the repository, and absent rather than wrong.** With no
// properties file the release build is unsigned, which cannot be installed —
// a loud failure. A release quietly signed with the debug key is a silent one,
// and it is the build somebody uploads.
val signingProps: Properties? = run {
    val path = System.getenv("KEEL_TV_KEYSTORE_PROPERTIES")
        ?: (System.getProperty("user.home") + "/keys/keel-tv.properties")
    val f = File(path)
    if (!f.exists()) return@run null
    val props = Properties()
    f.inputStream().use { props.load(it) }
    props
}

android {
    namespace = "uz.keel.tv"
    compileSdk = 36

    defaultConfig {
        // ⚠️ **The Expo build's id, on purpose.** This replaces that app on the
        // televisions it is already installed on; a new id would install beside
        // it and leave two Keels on a leanback home screen with nothing to tell
        // them apart — on a device driven by a remote from four metres away.
        applicationId = "uz.keel.tv"
        // ⚠️ 26, not 31. The blur that makes the glass is API 31+ and degrades
        // (see design/Glass.kt) — and the cheap Android TV boxes and sets sold
        // here run Android 8 to 11. A minSdk chosen for a visual effect would
        // lock out most of the walls this is for.
        minSdk = 26
        targetSdk = 36
        versionCode = 6
        // ⚠️ Continues the Expo build's numbering (it shipped versionCode 5,
        // 1.2.1): the store refuses an upload whose code is not higher than the
        // last one, and this is the same application id.
        versionName = "2.0.0"
        vectorDrawables { useSupportLibrary = true }
    }

    signingConfigs {
        if (signingProps != null) {
            create("release") {
                storeFile = file(signingProps.getProperty("storeFile"))
                storePassword = signingProps.getProperty("storePassword")
                keyAlias = signingProps.getProperty("keyAlias")
                keyPassword = signingProps.getProperty("keyPassword")
            }
        }
    }

    buildTypes {
        release {
            signingConfig = signingConfigs.findByName("release")
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }

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
    packaging { resources { excludes += "/META-INF/{AL2.0,LGPL2.1}" } }
}

dependencies {
    implementation(project(":design"))
    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.graphics)
    implementation(libs.compose.ui.tooling.preview)
    implementation(libs.compose.material3)
    implementation(libs.compose.material.icons)
    implementation(libs.activity.compose)
    implementation(libs.lifecycle.runtime.compose)
    implementation(libs.lifecycle.viewmodel.compose)
    implementation(libs.coroutines.android)
    implementation(libs.core.ktx)
    implementation(libs.core.splashscreen)

    implementation(libs.serialization.json)
    implementation(libs.ktor.client.okhttp)
    implementation(libs.ktor.client.content.negotiation)
    implementation(libs.ktor.serialization.json)

    implementation(libs.security.crypto)

    // The loop's video half. ⚠️ No `media3-ui`: see the note in the catalog.
    implementation(libs.media3.exoplayer)

    // ⚠️ **No Firebase.** Nothing is ever pushed to a television: it asks every
    // minute, which is also how being unpaired reaches it. A messaging library
    // here would be a Google dependency on a device that has to work when the
    // internet does not.

    testImplementation(libs.junit)
    debugImplementation(libs.compose.ui.tooling)
}
