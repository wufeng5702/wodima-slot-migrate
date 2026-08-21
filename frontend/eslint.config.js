// eslint.config.js
import js from '@eslint/js';
import ts from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';
import vue from 'eslint-plugin-vue';
import prettier from 'eslint-plugin-prettier';
import globals from 'globals';

export default [
  js.configs.recommended,

  // 1. TypeScript 配置（仅针对 .ts/.tsx）
  {
    files: ['**/*.ts', '**/*.tsx'],
    plugins: {
      '@typescript-eslint': ts,
    },
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        project: './tsconfig.json',
      },
    },
    rules: {
      ...ts.configs['recommended'].rules,

      // 关闭所有与格式相关的规则，让 Prettier 全权负责
      'semi': 'off',
      'comma-dangle': 'off',
      'quotes': 'off',
      'indent': 'off',
    },
  },
  {
    files: ['**/*.js', '**/*.ts', '**/*.tsx', '**/*.vue'],
    plugins: {
      '@typescript-eslint': ts,
    },
    rules: {
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': ['error', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        caughtErrorsIgnorePattern: '^_',
      }],
    },
  },

  // 2. Vue 配置（使用官方 flat/recommended）
  ...vue.configs['flat/recommended'],
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: tsParser,          // 让 vue-eslint-parser 将 TS 部分交给 @typescript-eslint/parser
        project: './tsconfig.json',
        extraFileExtensions: ['.vue'],
      },
    },
    rules: {
      'vue/multi-word-component-names': 'off', // 可选
      // 让 Prettier 全权处理 Vue 模板格式，避免和单行元素换行规则冲突导致 lint:fix 死循环
      'vue/singleline-html-element-content-newline': 'off',
      'vue/multiline-html-element-content-newline': 'off',
      'vue/html-closing-bracket-newline': 'off',
      'vue/html-indent': 'off',
      'vue/max-attributes-per-line': 'off',
      'vue/no-v-html': 'off',
      'vue/html-self-closing': 'off',
    },
  },

  // 3. Prettier 集成
  {
    plugins: {
      prettier,
    },
    rules: {
      'prettier/prettier': 'error',
    },
  },

  // 4. 全局忽略（替代 .eslintignore）
  {
    ignores: [
      '**/node_modules/**',
      '**/dist/**',
      '*.config.js',
      '*.config.ts',
      '**/*.d.ts',
    ],
  },

  // 5. 全局变量
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
        ...globals.es2021,
      },
    },
  },
];
