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

rootProject.name = "keel"
include(":app")

// ⚠️ **Included by path, the same directory the other five apps include.** One
// design system, not a sixth copy of it — see mobile/android-design.
include(":design")
project(":design").projectDir = file("../android-design")
