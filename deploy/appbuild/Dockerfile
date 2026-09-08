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
    # ⚠️ **The JVM must not try IPv6 here, and this cost an afternoon.** Docker's
    # resolver answers `dl.google.com` with AAAA records on this host while the
    # bridge has no IPv6 route out. `curl` notices and falls back in a
    # millisecond; Java does not — `sdkmanager` sits printing "Still waiting for
    # package manifests to be fetched remotely" for as long as anything lets it,
    # which for a build is until the forty-minute ceiling, holding the
    # one-at-a-time queue and producing nothing.
    #
    # ⚠️ It is a *hang*, not a failure: no error, no exit code, nothing in the
    # log but a warning that reads like patience.
    #
    # ⚠️ `JAVA_TOOL_OPTIONS` rather than `GRADLE_OPTS` because the tool that hung
    # was `sdkmanager`, which Gradle's options never reach. The JVM announces it
    # on stderr at every launch — more noise in a log the artifact path is no
    # longer read positionally from.
    JAVA_TOOL_OPTIONS="-Djava.net.preferIPv4Stack=true" \
    # ⚠️ **One heap, and it has to fit inside the container's limit.** The first
    # version asked for 2 GB for Gradle *and* 1 GB for a separate Kotlin daemon
    # inside a 3 GB container — two JVMs whose heaps alone equal the ceiling
    # before any overhead. The kernel killed one and Gradle reported "daemon
    # disappeared unexpectedly", which names neither memory nor the limit.
    #
    # The Kotlin compiler now runs in-process (see build.sh), so this is the
    # whole of it: 2 GB of heap inside a 3.5 GB container leaves room for the
    # JVM itself, R8 and the tooling around them.
    GRADLE_OPTS="-Dorg.gradle.jvmargs=-Xmx2g"

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
# ⚠️ **35.0.0 as well as 36.0.0, and 35 is the one that gets used.** AGP picks
# its own default build-tools version, and for 8.13 that is 35.0.0 — not the one
# `compileSdk = 36` suggests. With only 36 installed, every build downloaded 35
# at run time: minutes added to a nine-minute job, a network dependency the
# comment above says there should not be, and progress written to stderr that
# once got mistaken for a filename. Both are pinned so neither is a surprise.
RUN yes | sdkmanager --licenses > /dev/null \
    && sdkmanager --install \
        "platform-tools" \
        "platforms;android-36" \
        "build-tools;35.0.0" \
        "build-tools;36.0.0" \
    > /dev/null

WORKDIR /work
