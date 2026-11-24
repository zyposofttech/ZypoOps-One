import eslint from '@eslint/js';
import ts from 'typescript-eslint';
export default ts.config({ignores:['.work/**','node_modules/**','**/generated/**']}, eslint.configs.recommended, ...ts.configs.recommended, { files:['**/*.{ts,mjs}'], languageOptions:{globals:{Buffer:'readonly',process:'readonly',console:'readonly',URL:'readonly',structuredClone:'readonly',TextDecoder:'readonly'}} });
