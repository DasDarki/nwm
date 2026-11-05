import {useState, type ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useBaseUrl from '@docusaurus/useBaseUrl';
import Layout from '@theme/Layout';
import CodeBlock from '@theme/CodeBlock';
import styles from './index.module.css';

const installCommands = {
  linux: 'curl -fsSL https://nwm.dasdarki.de/install.sh | sh',
  windows: 'irm https://nwm.dasdarki.de/install.ps1 | iex',
};

type Platform = keyof typeof installCommands;

const features = [
  {
    icon: '🖥️',
    title: 'Servers in one command',
    text: 'Install, update and run nanos world servers through SteamCMD. Foreground or background, with logs, restarts and branch switching.',
    link: '/docs/servers/creating',
  },
  {
    icon: '📜',
    title: 'Recipes',
    text: 'Describe a whole server in a Recipefile: config, packages, assets, C modules, files, .env values and hooks. Share it through git, a URL or a gist.',
    link: '/docs/recipes/overview',
  },
  {
    icon: '🐳',
    title: 'Services next to your server',
    text: 'use postgres, use redis: nwm starts Docker containers with sane defaults, waits until they are ready and writes the connection URLs to .env.',
    link: '/docs/services',
  },
  {
    icon: '🔑',
    title: 'Secrets built in',
    text: 'Random or prompted passwords per server, stored outside your recipe and server folder, referenced as ${secret.name}.',
    link: '/docs/secrets',
  },
  {
    icon: '📦',
    title: 'Vault, git and NanosSharp',
    text: 'Install packages and asset packs from the vault, straight from git or from your recipe. Add C# scripting with NanosSharp in one step.',
    link: '/docs/servers/packages-and-assets',
  },
  {
    icon: '⚡',
    title: 'Editor and registry',
    text: 'An interactive recipe editor with completion from the nanos world store, and a git-backed registry to publish recipes as @you/name.',
    link: '/docs/recipes/editing',
  },
];

const terminalLines: {prompt?: boolean; text: string; tone?: 'muted' | 'success' | 'accent'}[] = [
  {prompt: true, text: 'nwm s create rp --recipe @dasdarki/roleplay'},
  {text: '✅ Fetching recipe @dasdarki/roleplay  v1.2.0', tone: 'success'},
  {text: '🔑 Stored 1 new secret(s) for rp: postgres_password', tone: 'muted'},
  {text: '✅ Installing nanos world server (public)  (12.4s)', tone: 'success'},
  {text: '✅ Installing vault packages  (3.1s)', tone: 'success'},
  {text: '✅ Starting services  (9.8s)', tone: 'success'},
  {text: '🎉 Server rp is ready', tone: 'accent'},
  {prompt: true, text: 'nwm s start rp -d'},
  {text: '✅ Starting rp  (2s)', tone: 'success'},
];

const recipeExample = `recipe @you/roleplay 1.0.0
description "Roleplay development server"

nanossharp
var port 7777

config discover.name "RP \${server.name}"
config discover.port \${port}

package sandbox enable
package rp-core git https://github.com/you/rp-core.git ref v2.1.0 enable

use postgres
use redis

secret steam_token prompt
env STEAM_TOKEN \${secret.steam_token}`;

function InstallBox(): ReactNode {
  const [platform, setPlatform] = useState<Platform>('linux');
  const [copied, setCopied] = useState(false);
  const command = installCommands[platform];
  const copy = async () => {
    await navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  return (
    <div className={styles.install}>
      <div className={styles.installTabs} role="tablist">
        {(Object.keys(installCommands) as Platform[]).map((key) => (
          <button
            key={key}
            type="button"
            role="tab"
            aria-selected={platform === key}
            className={clsx(styles.installTab, platform === key && styles.installTabActive)}
            onClick={() => setPlatform(key)}>
            {key === 'linux' ? 'Linux' : 'Windows'}
          </button>
        ))}
      </div>
      <div className={styles.installCommand}>
        <span className={styles.installPrompt}>{platform === 'linux' ? '$' : 'PS>'}</span>
        <code>{command}</code>
        <button type="button" className={styles.copy} onClick={copy} aria-label="Copy install command">
          {copied ? 'copied' : 'copy'}
        </button>
      </div>
    </div>
  );
}

function Terminal(): ReactNode {
  return (
    <div className={styles.terminal} aria-hidden="true">
      <div className={styles.terminalBar}>
        <span />
        <span />
        <span />
      </div>
      <div className={styles.terminalBody}>
        {terminalLines.map((line, index) => (
          <div key={index} className={clsx(styles.terminalLine, line.tone && styles[line.tone])} style={{animationDelay: `${index * 0.35}s`}}>
            {line.prompt && <span className={styles.terminalPrompt}>❯ </span>}
            {line.text}
          </div>
        ))}
      </div>
    </div>
  );
}

function Hero(): ReactNode {
  const logo = useBaseUrl('/img/logo.png');
  return (
    <header className={styles.hero}>
      <div className={styles.orbit} aria-hidden="true" />
      <div className={styles.orbitSecond} aria-hidden="true" />
      <div className={clsx('container', styles.heroInner)}>
        <div className={styles.heroText}>
          <img src={logo} alt="" className={styles.heroLogo} width={88} height={88} />
          <h1 className={styles.title}>nwm</h1>
          <p className={styles.subtitle}>nanos world manager</p>
          <p className={styles.tagline}>
            Create, run and update nanos world servers, describe them as recipes and bring their databases along.
            One cross platform CLI for Linux and Windows.
          </p>
          <InstallBox />
          <div className={styles.actions}>
            <Link className={clsx('button button--lg', styles.primary)} to="/docs/getting-started/quick-start">
              Quick start
            </Link>
            <Link className={clsx('button button--lg', styles.secondary)} to="/docs/intro">
              Read the docs
            </Link>
          </div>
        </div>
        <Terminal />
      </div>
    </header>
  );
}

function Features(): ReactNode {
  return (
    <section className={styles.section}>
      <div className="container">
        <h2 className={styles.sectionTitle}>Everything a nanos world server needs</h2>
        <div className={styles.features}>
          {features.map((feature) => (
            <Link key={feature.title} to={feature.link} className={styles.feature}>
              <span className={styles.featureIcon}>{feature.icon}</span>
              <h3>{feature.title}</h3>
              <p>{feature.text}</p>
            </Link>
          ))}
        </div>
      </div>
    </section>
  );
}

function RecipeShowcase(): ReactNode {
  return (
    <section className={clsx(styles.section, styles.showcase)}>
      <div className={clsx('container', styles.showcaseInner)}>
        <div className={styles.showcaseText}>
          <h2 className={styles.sectionTitle}>A whole server in one file</h2>
          <p>
            A <code>Recipefile</code> lists everything your server needs. nwm installs the server, pulls packages from
            the vault or git, starts Postgres and Redis, generates their passwords and writes <code>DATABASE_URL</code>{' '}
            and <code>REDIS_URL</code> to the server&apos;s <code>.env</code>.
          </p>
          <p>
            Write it by hand, with <code>nwm recipe edit</code> and tab completion, or one line at a time with{' '}
            <code>nwm recipe add</code>.
          </p>
          <Link className={clsx('button', styles.secondary)} to="/docs/recipes/recipefile">
            Recipefile reference
          </Link>
        </div>
        <div className={styles.showcaseCode}>
          <CodeBlock language="recipefile" title="Recipefile">
            {recipeExample}
          </CodeBlock>
        </div>
      </div>
    </section>
  );
}

export default function Home(): ReactNode {
  return (
    <Layout title="nanos world manager" description="Create, run and update nanos world servers, describe them as recipes and bring their services along.">
      <Hero />
      <main>
        <Features />
        <RecipeShowcase />
      </main>
    </Layout>
  );
}
