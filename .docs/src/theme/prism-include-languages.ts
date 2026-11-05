import siteConfig from '@generated/docusaurus.config';
import type * as PrismNamespace from 'prismjs';

const recipefileKeywords = [
  'recipe', 'description', 'author', 'keywords', 'branch', 'nanossharp', 'arg', 'var', 'config', 'cfg',
  'package', 'pkg', 'assets', 'asset', 'module', 'mod', 'file', 'use', 'require_service', 'requires',
  'require', 'service', 'svc', 'secret', 'env', 'hook',
];

const recipefileOptions = [
  'enable', 'git', 'ref', 'subdir', 'path', 'linux', 'windows', 'template', 'port', 'env', 'volume', 'ready',
  'nowait', 'command', 'as', 'random', 'prompt', 'in', 'on', 'off', 'public', 'bleeding-edge',
];

function registerRecipefile(prism: typeof PrismNamespace): void {
  prism.languages.recipefile = {
    comment: {pattern: /(^|\s)#.*/m, lookbehind: true, greedy: true},
    string: {
      pattern: /"(?:\\.|[^"\\])*"|'[^']*'/,
      greedy: true,
      inside: {
        variable: /\$\{[\w.]+\}/,
      },
    },
    variable: /\$\{[\w.]+\}/,
    keyword: {pattern: new RegExp(`^(\\s*)(?:${recipefileKeywords.join('|')})(?=\\s|$)`, 'm'), lookbehind: true},
    builtin: {pattern: new RegExp(`(\\s)(?:${recipefileOptions.join('|')})(?=\\s|$)`), lookbehind: true},
    scope: {pattern: /@[\w-]+\/[\w-]+/, alias: 'class-name'},
    url: {pattern: /\b(?:https?|git|file):\/\/\S+|\bgithub:\S+|\bgist:\S+/, alias: 'url'},
    boolean: /\b(?:true|false)\b/,
    number: /(?<![\w./:-])\d+(?:\.\d+)?(?![\w./:-])/,
    flag: {pattern: /(\s)--?[\w-]+/, lookbehind: true, alias: 'attr-name'},
  };
}

export default function prismIncludeLanguages(PrismObject: typeof PrismNamespace): void {
  const {
    themeConfig: {prism},
  } = siteConfig;
  const {additionalLanguages} = prism as {additionalLanguages: string[]};
  const PrismBefore = globalThis.Prism;
  globalThis.Prism = PrismObject;
  additionalLanguages.forEach((language) => {
    require(`prismjs/components/prism-${language}`);
  });
  registerRecipefile(PrismObject);
  delete (globalThis as {Prism?: unknown}).Prism;
  if (typeof PrismBefore !== 'undefined') {
    globalThis.Prism = PrismObject;
  }
}
