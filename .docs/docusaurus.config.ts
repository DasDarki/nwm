import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const repository = 'https://github.com/DasDarki/nwm';

const darkCodeTheme = {
  ...prismThemes.oceanicNext,
  plain: {...prismThemes.oceanicNext.plain, backgroundColor: '#021c26', color: '#d5eef2'},
};

const config: Config = {
  title: 'nwm',
  tagline: 'The nanos world manager. Servers, recipes, packages and services from one CLI.',
  favicon: 'img/favicon.png',
  url: 'https://nwm.dasdarki.de',
  baseUrl: '/',
  trailingSlash: false,
  organizationName: 'DasDarki',
  projectName: 'nwm',
  onBrokenLinks: 'throw',
  onBrokenAnchors: 'throw',
  markdown: {
    format: 'detect',
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },
  headTags: [
    {tagName: 'link', attributes: {rel: 'apple-touch-icon', href: '/img/apple-touch-icon.png'}},
    {tagName: 'meta', attributes: {name: 'theme-color', content: '#011822'}},
  ],
  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: 'docs',
          editUrl: `${repository}/edit/main/.docs/`,
          showLastUpdateTime: false,
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],
  themes: [
    [
      '@easyops-cn/docusaurus-search-local',
      {
        hashed: true,
        indexBlog: false,
        docsRouteBasePath: '/docs',
        highlightSearchTermsOnTargetPage: true,
        explicitSearchResultPath: true,
      },
    ],
  ],
  themeConfig: {
    image: 'img/social-card.png',
    colorMode: {
      defaultMode: 'dark',
      respectPrefersColorScheme: false,
    },
    navbar: {
      title: 'nwm',
      logo: {
        alt: 'nwm logo',
        src: 'img/logo.png',
      },
      items: [
        {type: 'docSidebar', sidebarId: 'docs', position: 'left', label: 'Docs'},
        {to: '/docs/recipes/recipefile', label: 'Recipefile', position: 'left'},
        {to: '/docs/reference/cli', label: 'CLI reference', position: 'left'},
        {href: 'https://github.com/DasDarki/nwm-registry', label: 'Registry', position: 'right'},
        {href: repository, label: 'GitHub', position: 'right'},
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {label: 'Installation', to: '/docs/getting-started/installation'},
            {label: 'Quick start', to: '/docs/getting-started/quick-start'},
            {label: 'Recipefile reference', to: '/docs/recipes/recipefile'},
            {label: 'CLI reference', to: '/docs/reference/cli'},
          ],
        },
        {
          title: 'Project',
          items: [
            {label: 'GitHub', href: repository},
            {label: 'Releases', href: `${repository}/releases`},
            {label: 'Recipe registry', href: 'https://github.com/DasDarki/nwm-registry'},
          ],
        },
        {
          title: 'nanos world',
          items: [
            {label: 'nanos world', href: 'https://nanos-world.com'},
            {label: 'nanos world docs', href: 'https://docs.nanos-world.com'},
            {label: 'NanosSharp', href: 'https://nanossharp.dasdarki.de'},
          ],
        },
      ],
      copyright: `nwm is a community project and not affiliated with nanos world. Built by DasDarki.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: darkCodeTheme,
      additionalLanguages: ['bash', 'powershell', 'toml', 'json'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
