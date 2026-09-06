pluginManagement {
    repositories {
        google {
            content {
                includeGroupByRegex("com\\.android.*")
                includeGroupByRegex("com\\.google.*")
                includeGroupByRegex("androidx.*")
            }
        }
        mavenCentral()
        gradlePluginPortal()
    }
}
dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "keel-tv"
include(":app")

// ⚠️ **The same directory the two phone applications include, by path.** A
// television is not a phone and almost nothing on this screen is shaped like a
// phone screen — but the orange, the glass, the near-black ground and the rule
// that turns "osh" into a server address are the product, not the form factor.
// A copy of them here would be the fourth, and the fourth is the one that drifts
// while nobody is looking at a screen hanging above head height.
include(":design")
project(":design").projectDir = file("../android-design")
