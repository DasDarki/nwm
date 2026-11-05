import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docs: [
    'intro',
    {
      type: 'category',
      label: 'Getting started',
      collapsed: false,
      items: ['getting-started/installation', 'getting-started/quick-start', 'getting-started/concepts'],
    },
    {
      type: 'category',
      label: 'Servers',
      items: [
        'servers/creating',
        'servers/running',
        'servers/configuration',
        'servers/packages-and-assets',
        'servers/nanossharp',
        'servers/c-modules',
      ],
    },
    {
      type: 'category',
      label: 'Recipes',
      items: [
        'recipes/overview',
        'recipes/recipefile',
        'recipes/variables',
        'recipes/editing',
        'recipes/applying',
        'recipes/sources',
        'recipes/recipe-toml',
      ],
    },
    'services',
    'secrets',
    {
      type: 'category',
      label: 'Registry',
      items: ['registry/using', 'registry/publishing', 'registry/how-it-works'],
    },
    {
      type: 'category',
      label: 'Reference',
      items: ['reference/cli', 'reference/environment', 'reference/files'],
    },
    'troubleshooting',
  ],
};

export default sidebars;
