# ---- The machine that builds a restaurant's app ----
#
# ⚠️ **An image rather than packages on the host.** That box runs every tenant's
# container; a JDK, an Android SDK and a Gradle cache installed beside them are
# three things that can be upgraded out from under a build, and one apt upgrade
# away from a build failure nobody can reproduce. In here the toolchain is a
# fact of the image, and the image has a tag.
#
# ⚠️ **The SDK is downloaded once, at build time, and pinned.** A build that
# fetched a platform on first use would be a build that fails the first time
# Google moves something — on the morning somebody is waiting for an APK.
FROM eclipse-temurin:17-jdk-jammy

ENV ANDROID_HOME=/opt/android-sdk \
    ANDROID_SDK_ROOT=/opt/android-sdk \
    DEBIAN_FRONTEND=noninteractive \
    # ⚠️ **The container's clock, or the stamped filename lies.** Without a zone
    # a container is UTC whatever the host is set to, so a build finished at ten
    # past three is filed as 10:10 — five hours off, in the one place somebody
    # looks to tell two builds apart. The same trap `cmd/server` carries
    # `time/tzdata` for, arriving in a different room.
    TZ=Asia/Tashkent \
    # ⚠️ Gradle runs with no daemon (see build.sh) but still needs a heap that
    # fits: 2 GB leaves room on a 7 GB box that is also running the restaurants.
    GRADLE_OPTS="-Dorg.gradle.jvmargs=-Xmx2g -Dkotlin.daemon.jvm.options=-Xmx1g"

RUN apt-get update && apt-get install -y --no-install-recommends \
        curl unzip jq git ca-certificates tzdata util-linux \
        python3 python3-pil \
    && rm -rf /var/lib/apt/lists/*

# The command-line tools, which are the only part of the SDK that can install
# the rest.
ARG CMDLINE_VERSION=11076708
RUN mkdir -p "$ANDROID_HOME/cmdline-tools" \
    && curl -fsSL -o /tmp/tools.zip \
        "https://dl.google.com/android/repository/commandlinetools-linux-${CMDLINE_VERSION}_latest.zip" \
    && unzip -q /tmp/tools.zip -d "$ANDROID_HOME/cmdline-tools" \
    && mv "$ANDROID_HOME/cmdline-tools/cmdline-tools" "$ANDROID_HOME/cmdline-tools/latest" \
    && rm /tmp/tools.zip

ENV PATH="$PATH:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools"

# ⚠️ **Pinned to what `app/build.gradle.kts` asks for.** A compileSdk the image
# does not have is a download at build time; a build-tools version drifting from
# the one Gradle picks is a rebuild of nothing anybody asked for.
RUN yes | sdkmanager --licenses > /dev/null \
    && sdkmanager --install \
        "platform-tools" \
        "platforms;android-36" \
        "build-tools;36.0.0" \
    > /dev/null

WORKDIR /work
