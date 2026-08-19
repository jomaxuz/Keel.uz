// ⚠️ `.mjs`, not `.js`: this package is `"type": "module"`, so a CommonJS
// `module.exports` here fails the build with an error about file extensions
// rather than about PostCSS.
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
};
