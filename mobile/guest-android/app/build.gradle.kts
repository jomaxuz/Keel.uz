// ⚠️ Imported rather than written out: inside a Gradle Kotlin DSL script `java`
// resolves to the Java plugin's extension, not to the package, so
// `java.util.Properties` does not compile and the error names `util`.
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
}

// ---- What makes this build one restaurant's app rather than another's ----
//
// ⚠️ **Read from one file, and the build fails loudly without it.** The
// alternative — a default baked in here — is a pipeline bug that produces a
// perfectly working application pointed at the wrong restaurant, signed, named
// and uploaded before anybody notices. A missing file is a stopped build; a
// silently wrong server is a guest ordering from somebody else's kitchen.
val brand: Properties = Properties().apply {
    val f = file("brand.properties")
    require(f.exists()) {
        "app/brand.properties yo'q — brendlanmagan build. Qarang: mobile/guest-android/README.md"
    }
    f.inputStream().use { load(it) }
}

fun brandOf(key: String): String = requireNotNull(brand.getProperty(key)) {
    "brand.properties da '$key' yo'q"
}

// Where this build is signed from.
//
// ⚠️ **One keystore per restaurant, and it can never be rotated**: an
// application on the store signed by a lost key can never be updated again. The
// pipeline generates it once, on the first build, and never regenerates it —
// see control/internal/appbuild.
//
// ⚠️ **Absent rather than wrong.** With no properties file the release build is
// unsigned, which cannot be installed: a loud failure. A release quietly signed
// with the debug key is a silent one, and it is the build somebody uploads.
val signingProps: Properties? = run {
    val path = System.getenv("KEEL_GUEST_KEYSTORE_PROPERTIES") ?: return@run null
    val f = File(path)
    if (!f.exists()) return@run null
    Properties().apply { f.inputStream().use { load(it) } }
}

android {
    namespace = "uz.keel.guest"
    compileSdk = 36

    defaultConfig {
        applicationId = brandOf("brand.applicationId")
        // ⚠️ 26, not 31. The blur that makes the glass is API 31+ and degrades
        // gracefully (design/Glass.kt). This application is installed by
        // whoever eats at the restaurant, which includes the oldest phone in
        // the city — a minSdk chosen for a visual effect would turn guests away
        // at the store page.
        minSdk = 26
        targetSdk = 36
        versionCode = brandOf("brand.versionCode").toInt()
        versionName = brandOf("brand.versionName")
        vectorDrawables { useSupportLibrary = true }

        resValue("string", "app_name", brandOf("brand.appName"))
        // ⚠️ **Through BuildConfig rather than a generated Kotlin file.** Both
        // work; this one cannot be edited by hand and then quietly disagree
        // with `brand.properties`, because there is no file to edit.
        buildConfigField("String", "SERVER_URL", "\"${brandOf("brand.serverUrl")}\"")
        buildConfigField("String", "BRAND_ACCENT", "\"${brandOf("brand.accent")}\"")

        // ⚠️ **The maps key goes in the manifest, not into a runtime field.**
        // The Android SDK reads it before any Kotlin runs, so unlike the name
        // and the colour it genuinely cannot come from `/restaurant` at launch —
        // which is the reason a per-restaurant build was already the model.
        //
        // ⚠️ Empty is allowed and gives a grey grid rather than a crash: a
        // restaurant that does not deliver never opens this screen, and refusing
        // to build their app over a key they do not need would be the wrong
        // failure.
        manifestPlaceholders["mapsApiKey"] = brand.getProperty("brand.mapsKey").orEmpty()
        // ⚠️ Also as a field, so the picker can say "the map is not set up"
        // instead of drawing a grey grid nobody can act on.
        buildConfigField(
            "String",
            "MAPS_KEY",
            "\"${brand.getProperty("brand.mapsKey").orEmpty()}\"",
        )
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
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
        debug {
            // ⚠️ So a developer's build installs beside a real restaurant's on
            // the same phone. Without it, testing against a live tenant
            // uninstalls the guest's own app and takes their session with it.
            applicationIdSuffix = ".debug"
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
    implementation(libs.core.ktx)
    implementation(libs.core.splashscreen)

    implementation(libs.serialization.json)
    implementation(libs.ktor.client.okhttp)
    implementation(libs.ktor.client.content.negotiation)
    implementation(libs.ktor.serialization.json)

    // Dish photographs, which is most of what this screen is.
    implementation(libs.coil.compose)
    implementation(libs.coil.network.okhttp)

    implementation(libs.security.crypto)

    // ---- The address picker ----
    //
    // ⚠️ **Google Maps, and the cost note in CLAUDE.md does not apply here.**
    // That warning is about the **web** APIs — MapGL, the Maps JavaScript API,
    // Static Maps — which bill per map load. The Android SDK's map display is
    // not billed at all, so the one reason the site defaults to 2GIS does not
    // reach this screen.
    //
    // ⚠️ **Geocoding stays on Nominatim** (`lib/geocode.ts` on the web side, the
    // same service here): the Places and Geocoding APIs *are* billed, and the
    // site already made this split for the same reason. The map draws; the
    // search does not go to Google.
    //
    // ⚠️ **One provider, resolved at build time.** Shipping all three SDKs so a
    // restaurant could switch at runtime would add tens of megabytes to every
    // guest's download for a choice that never changes once made.
    implementation(libs.play.services.maps)
    implementation(libs.maps.compose)
    implementation(libs.play.services.location)

    testImplementation(libs.junit)
    debugImplementation(libs.compose.ui.tooling)
}
