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

rootProject.name = "keel-courier"
include(":app")

// ⚠️ **The same directory the other three include, by path.** The glass, the
// orange, the near-black and the rule that turns "osh" into a server address are
// the product; a copy here would be the fourth, and a courier's phone is where a
// drift would be least visible and most annoying.
include(":design")
project(":design").projectDir = file("../android-design")
