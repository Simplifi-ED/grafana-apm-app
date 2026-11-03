// Ambient declaration for side-effect CSS imports.
// TypeScript 6.0 requires a module declaration for these; the scaffolded
// .config/types/bundler-rules.d.ts only covers image and font assets. Webpack
// handles the actual bundling of these stylesheets at build time.
declare module '*.css';
