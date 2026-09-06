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

rootProject.name = "keel-team"
include(":app")

// ⚠️ **The same directory the other three include, by path.** The glass, the
// orange, the near-black and the rule that turns "osh" into a server address are
// the product. This is the fourth application to include it and the last: a
// fifth copy would be the one that drifts, on the phone every single person in
// a restaurant carries.
include(":design")
project(":design").projectDir = file("../android-design")
