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

rootProject.name = "keel-guest"
include(":app")

// ⚠️ **The same directory the five staff applications include, by path.** The
// glass, the near-black and the three languages are the product; a sixth copy
// would be the one that drifts — and this is the application a **guest**
// installs, so the drift would be visible to somebody who is not paid to
// tolerate it.
include(":design")
project(":design").projectDir = file("../android-design")
