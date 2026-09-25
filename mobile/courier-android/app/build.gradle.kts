// ⚠️ Imported rather than written out in full: inside a Gradle Kotlin DSL
// script `java` resolves to the Java plugin's extension, not to the package, so
// `java.util.Properties` does not compile and the error names `util`.
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.google.services)
}

// Where this build is signed from.
//
// ⚠️ **Its own key, like the other three.** A signing key cannot be rotated: an
// application on the store signed by a lost key can never be updated again. One
// key across four applications makes them one blast radius, and this one is
// handed over on its own schedule — a courier's phone is replaced, sold and
// passed on far more often than a manager's.
//
// ⚠️ **Read from outside the repository, and absent rather than wrong.** With no
// properties file the release build is unsigned, which cannot be installed — a
// loud failure. A release quietly signed with the debug key is a silent one, and
// it is the build somebody uploads.
val signingProps: Properties? = run {
    val path = System.getenv("KEEL_COURIER_KEYSTORE_PROPERTIES")
        ?: (System.getProperty("user.home") + "/keys/keel-courier.properties")
    val f = File(path)
    if (!f.exists()) return@run null
    val props = Properties()
    f.inputStream().use { props.load(it) }
    props
}

// What this build calls itself.
//
// ⚠️ **The platform's one number, read from the repo-root `VERSION`.** The
// dashboard, the console and the till all say that number, and this app used to
// carry its own ("1.0.0" here, "2.0.0" in a constant beside it) — three answers
// to "which version?" on one support call. `scripts/set-version.sh` writes
// `VERSION`, so a release moves this too without anyone remembering the app.
// The code only grows (major·10000 + minor·100 + patch), which is what Android
// needs to install a build over the last one.
val keelVersion: String = rootDir.resolve("../../VERSION").readText().trim().removePrefix("v")
val keelVersionCode: Int = keelVersion.split(".").map { it.toInt() }
    .let { (major, minor, patch) -> major * 10000 + minor * 100 + patch }

android {
    namespace = "uz.keel.courier"
    compileSdk = 36

    defaultConfig {
        // ⚠️ **The Expo build's id, on purpose.** This replaces that app on the
        // phones it is already on; a new id would install beside it and leave a
        // courier with two Keels on a home screen and no way to tell which one
        // the dispatcher can see.
        applicationId = "uz.keel.courier"
        // ⚠️ 26, not 31. The blur that makes the glass is API 31+ and degrades
        // (see design/Glass.kt) — and the phones couriers here carry run
        // Android 8 to 11. A minSdk chosen for a visual effect would lock out
        // most of the people this app is for.
        minSdk = 26
        targetSdk = 36
        versionCode = keelVersionCode
        versionName = keelVersion
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
    implementation(libs.coroutines.play.services)
    implementation(libs.core.ktx)
    implementation(libs.core.splashscreen)

    implementation(libs.serialization.json)
    implementation(libs.ktor.client.okhttp)
    implementation(libs.ktor.client.content.negotiation)
    implementation(libs.ktor.serialization.json)

    implementation(libs.security.crypto)

    // Where the courier is. ⚠️ The fused provider rather than `LocationManager`:
    // it is what merges GPS, wifi and cell into one answer and what stops a
    // phone burning a battery holding a satellite fix in a delivery van.
    implementation(libs.play.services.location)

    implementation(platform(libs.firebase.bom))
    implementation(libs.firebase.messaging)

    testImplementation(libs.junit)
    debugImplementation(libs.compose.ui.tooling)
}
