// What turns an ordinary Android build into one a television will run.
//
// ⚠️ **Without this the app installs and is invisible.** An Android TV home
// screen lists only activities that declare `LEANBACK_LAUNCHER`; a normal
// `LAUNCHER` activity is what a phone shows and what a television ignores. The
// APK sideloads fine, reports success, and then there is nothing to open —
// which reads as a broken build rather than as a missing line of manifest.
//
// ⚠️ **And it must still install on a phone**, because that is where this is
// developed and tested: `required="false"` on both features. Marking leanback
// required is the tidy answer and it makes every phone say "app not compatible"
// — including the one belonging to whoever is trying to reproduce a bug.
//
// The three declarations, and why each is here:
//   - leanback (not required): tells the system this is a TV app.
//   - touchscreen (not required): a television has none, and the default is
//     `true` — Play Store filters the app off every TV without it.
//   - a banner: the leanback launcher draws one for each app. Missing, the
//     entry is blank on some launchers and the listing is refused by review.

const { AndroidConfig, withAndroidManifest } = require("expo/config-plugins");

const LEANBACK = "android.software.leanback";
const TOUCHSCREEN = "android.hardware.touchscreen";
const LEANBACK_LAUNCHER = "android.intent.category.LEANBACK_LAUNCHER";

module.exports = function withAndroidTV(config) {
  return withAndroidManifest(config, (cfg) => {
    const manifest = cfg.modResults.manifest;

    manifest["uses-feature"] = manifest["uses-feature"] ?? [];
    const feature = (name) => {
      const rows = manifest["uses-feature"];
      const found = rows.find((f) => f.$?.["android:name"] === name);
      if (found) {
        found.$["android:required"] = "false";
        return;
      }
      rows.push({ $: { "android:name": name, "android:required": "false" } });
    };
    feature(LEANBACK);
    feature(TOUCHSCREEN);

    const app = AndroidConfig.Manifest.getMainApplicationOrThrow(cfg.modResults);
    // ⚠️ On the application as well as on the activity: some launchers read one
    // and some read the other, and an app with no banner is a blank tile.
    app.$["android:banner"] = "@mipmap/ic_launcher";

    const activity = AndroidConfig.Manifest.getMainActivityOrThrow(cfg.modResults);
    activity.$["android:banner"] = "@mipmap/ic_launcher";
    // A television is landscape and cannot be turned. Saying so stops the app
    // being recreated by a rotation event some sets send at boot.
    activity.$["android:screenOrientation"] = "landscape";

    for (const filter of activity["intent-filter"] ?? []) {
      const isLauncher = (filter.category ?? []).some(
        (c) => c.$?.["android:name"] === "android.intent.category.LAUNCHER",
      );
      if (!isLauncher) continue;
      const already = (filter.category ?? []).some(
        (c) => c.$?.["android:name"] === LEANBACK_LAUNCHER,
      );
      if (!already) {
        filter.category.push({ $: { "android:name": LEANBACK_LAUNCHER } });
      }
    }

    return cfg;
  });
};
