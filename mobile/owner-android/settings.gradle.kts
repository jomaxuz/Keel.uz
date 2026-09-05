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

rootProject.name = "keel-owner"
include(":app")

// ⚠️ **Included by path, and it lives outside this project on purpose.** The
// owner application includes the very same directory from its own build, which
// is what makes it one copy rather than two that look alike for a while.
include(":design")
project(":design").projectDir = file("../android-design")
