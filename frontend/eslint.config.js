// eslint.config.js
import js from "@eslint/js";
import svelte from "eslint-plugin-svelte";
import globals from "globals";
import ts from "typescript-eslint";
import svelteConfig from "./svelte.config.js";

export default ts.config(
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs.recommended,
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
  },
  {
    files: ["**/*.svelte", "**/*.svelte.ts", "**/*.svelte.js"],
    // See more details at: https://typescript-eslint.io/packages/parser/
    languageOptions: {
      parserOptions: {
        projectService: true,
        extraFileExtensions: [".svelte"], // Add support for additional file extensions, such as .svelte
        parser: ts.parser,
        // Specify a parser for each language, if needed:
        // parser: {
        //   ts: ts.parser,
        //   js: espree,    // Use espree for .js files (add: import espree from 'espree')
        //   typescript: ts.parser
        // },

        // We recommend importing and specifying svelte.config.js.
        // By doing so, some rules in eslint-plugin-svelte will automatically read the configuration and adjust their behavior accordingly.
        // While certain Svelte settings may be statically loaded from svelte.config.js even if you don’t specify it,
        // explicitly specifying it ensures better compatibility and functionality.
        svelteConfig,
      },
    },
  },
  {
    rules: {
      // Disallow trailing spaces in Svelte files.
      "svelte/no-trailing-spaces": "error",

      // Enforce spacing inside HTML comments (e.g., <!-- comment --> instead of <!--comment-->).
      "svelte/spaced-html-comment": "error",

      // Enforce use of shorthand syntax for HTML attributes when possible (e.g., <input disabled> instead of <input disabled={true}>).
      "svelte/shorthand-attribute": "error",

      // Enforce use of shorthand syntax for directives (e.g., use:action instead of use:action={action}).
      "svelte/shorthand-directive": "error",

      // Enforce consistent spacing inside mustache interpolations (e.g., {{ value }}).
      "svelte/mustache-spacing": "error",

      // Disallow spaces around equal signs in HTML attributes (e.g., <div class="box"> instead of <div class = "box">).
      "svelte/no-spaces-around-equal-signs-in-attribute": "error",

      // Enforce consistent use of quotes in HTML attributes.
      "svelte/html-quotes": [
        "error",
        {
          // Prefer double quotes in attributes (e.g., class="value").
          prefer: "double",
          dynamic: {
            // Allow unquoted dynamic values unless invalid in HTML.
            quoted: false,
            avoidInvalidUnquotedInHTML: false,
          },
        },
      ],

      // Enforce consistent spacing before closing brackets in HTML tags.
      "svelte/html-closing-bracket-spacing": [
        "error",
        {
          // No space before closing bracket of start tag (e.g., <div> not <div >).
          startTag: "never",
          // No space before closing bracket of end tag (e.g., </div> not </div >).
          endTag: "never",
          // Always space before self-closing tags (e.g., <img />).
          selfClosingTag: "always",
        },
      ],

      // Prefer `const` declarations where variables are not reassigned.
      "svelte/prefer-const": [
        "error",
        {
          // Apply even to destructured variables.
          destructuring: "any",
          // Exclude reactive declarations (e.g., $props, $derived).
          excludedRunes: ["$props", "$derived"],
        },
      ],

      // Prefer destructuring store props for clarity (e.g., const { foo } = $store instead of $store.foo).
      "svelte/prefer-destructured-store-props": "error",

      // Enforce specifying a `type` attribute on all `<button>` elements.
      "svelte/button-has-type": [
        "error",
        {
          // Require type for <button>, <button type="submit">, and <button type="reset">.
          button: true,
          submit: true,
          reset: true,
        },
      ],

      // Can't use inline style
      "svelte/no-inline-styles": [
        "error",
        {
          allowTransitions: true,
        },
      ],

      // Require semicolons at the end of statements.
      semi: ["error", "always"],

      // Enforce double quotes for string literals in TypeScript and JavaScript code.
      quotes: ["error", "double"],

      // Disallow unused variables in TypeScript.
      "@typescript-eslint/no-unused-vars": ["error"],

      // Not allowed to use any
      "@typescript-eslint/no-explicit-any": ["error"],

      // Warn when `console.log` or similar are used (useful for keeping production code clean).
      "no-console": "warn",
    },
  }
);
