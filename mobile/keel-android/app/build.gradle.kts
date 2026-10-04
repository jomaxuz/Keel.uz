// ⚠️ Imported rather than written out in full: inside a Gradle Kotlin DSL
// script `java` resolves to the Java plugin's extension, not to the package, so
// `java.util.Properties` does not compile and the error names `util`.
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.ksp)
    alias(libs.plugins.google.services) apply false
}

// ⚠️ **Firebase only once it knows this app.** Push needs `google-services.json`
// with a client for `uz.keel.app` — an app registered in the Firebase console,
// which is a step taken there and not here. Until it is, the plugin would fail
// the whole build ("No matching client found"), so it is applied only when the
// file is present and names this package. Without it the app builds and runs;
// the settings screens say push is not configured (`PushState.NoProject`),
// which is the honest state.
val firebaseReady: Boolean = run {
    val f = file("google-services.json")
    f.exists() && f.readText().contains("\"uz.keel.app\"")
}
if (firebaseReady) {
    apply(plugin = libs.plugins.google.services.get().pluginId)
}

// Where this build is signed from — outside the repository, as for every other
// Keel app. A checkout without the file builds unsigned rather than failing.
val signingProps: Properties? = run {
    val path = System.getenv("KEEL_KEYSTORE_PROPERTIES")
        ?: (System.getProperty("user.home") + "/keys/keel-app.properties")
    val f = File(path)
    if (!f.exists()) return@run null
    val props = Properties()
    f.inputStream().use { props.load(it) }
    props
}

// The platform's one number, read from the repo-root `VERSION`.
val keelVersion: String = rootDir.resolve("../../VERSION").readText().trim().removePrefix("v")
val keelVersionCode: Int = keelVersion.split(".").map { it.toInt() }
    .let { (major, minor, patch) -> major * 10000 + minor * 100 + patch }

android {
    namespace = "uz.keel.app"
    compileSdk = 36

    defaultConfig {
        // ⚠️ **Its own id, beside the four apps it unites — not instead of
        // them.** Keel Waiter, Owner, Courier and Team stay installed and keep
        // working while restaurants move over; a shared id would replace one of
        // them on update and strand whoever still needed it.
        applicationId = "uz.keel.app"
        // 26, like the others: the phones sold into restaurants here run
        // Android 8 to 11.
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
            freeCompilerArgs.add("-opt-in=androidx.compose.foundation.layout.ExperimentalLayoutApi")
        }
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }
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

    implementation(libs.coil.compose)
    implementation(libs.coil.network.okhttp)

    implementation(libs.security.crypto)
    implementation(libs.datastore.preferences)
    // The waiter's offline outbox.
    implementation(libs.room.runtime)
    implementation(libs.room.ktx)
    ksp(libs.room.compiler)

    implementation(libs.play.services.location)
    // The team's stock count and marking scanner.
    implementation(libs.camera.core)
    implementation(libs.camera.camera2)
    implementation(libs.camera.lifecycle)
    implementation(libs.camera.view)
    implementation(libs.mlkit.barcode)

    implementation(platform(libs.firebase.bom))
    implementation(libs.firebase.messaging)

    testImplementation(libs.junit)
    debugImplementation(libs.compose.ui.tooling)
}
