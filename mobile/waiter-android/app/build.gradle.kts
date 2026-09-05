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
    alias(libs.plugins.google.services)
}

// Where this build is signed from.
//
// ⚠️ **Outside the repository, and read rather than committed.** The keystore
// and its password are the one secret here that cannot be rotated: an app on the
// store signed by a lost key can never be updated again — a new key is a new
// application, with nobody's installs and none of its reviews. So the file lives
// in `~/keys`, this reads a path to it, and a checkout on another machine simply
// builds unsigned rather than failing.
val signingProps: Properties? = run {
    val path = System.getenv("KEEL_KEYSTORE_PROPERTIES")
        ?: (System.getProperty("user.home") + "/keys/keel-waiter.properties")
    val f = File(path)
    if (!f.exists()) return@run null
    val props = Properties()
    f.inputStream().use { props.load(it) }
    props
}

android {
    namespace = "uz.keel.waiter"
    compileSdk = 36

    defaultConfig {
        // ⚠️ **Same id as the Expo build on purpose.** This replaces that app on
        // the same phones; a new id would install beside it and a waiter would
        // have two Keels on the home screen and no way to tell which one the
        // kitchen is talking to.
        applicationId = "uz.keel.waiter"
        // ⚠️ 26, not 31. The blur that makes the glass is API 31+ and has a
        // fallback (see Glass.kt) — but the phones sold into restaurants here
        // run Android 8 to 11, and a minSdk chosen for a visual effect would
        // lock out the customers this app is for.
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "1.0.0"
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
            // ⚠️ Absent rather than wrong when the properties file is not there:
            // an unsigned APK cannot be installed, which is a loud failure. A
            // release quietly signed by the *debug* key is a silent one — and it
            // is the build somebody uploads.
            signingConfig = signingConfigs.findByName("release")
            isMinifyEnabled = true
            // ⚠️ Resource shrinking as well as code: on the Expo build this was
            // the half people forget, and most of a package is libraries'
            // images and translations rather than code.
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }

    // ⚠️ JDK 17. The machine currently has 11 only — see README, "Nima yetishmayapti".
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlin {
        compilerOptions {
            jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
            // ⚠️ Opted in once, here, rather than annotated at thirty call
            // sites. `ModalBottomSheet` is the only experimental API this app
            // uses, and the annotation is viral: it propagates up through every
            // composable that transitively reaches the sheet — which on the
            // check screen is the whole screen. One line and a reason beats
            // thirty annotations nobody can tell apart from meaningful ones.
            freeCompilerArgs.add("-opt-in=androidx.compose.material3.ExperimentalMaterial3Api")
        }
    }

    buildFeatures { compose = true }
    packaging { resources { excludes += "/META-INF/{AL2.0,LGPL2.1}" } }
}

dependencies {
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
    implementation(libs.room.runtime)
    implementation(libs.room.ktx)
    ksp(libs.room.compiler)

    implementation(libs.play.services.location)
    implementation(platform(libs.firebase.bom))
    implementation(libs.firebase.messaging)

    debugImplementation(libs.compose.ui.tooling)
}
