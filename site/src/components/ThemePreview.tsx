import React, { useState } from 'react';
import { Check, Sun, Moon, Monitor } from 'lucide-react';

interface ThemeInfo {
  id: string;
  name: string;
  tag: string;
  bg: string;
  fg: string;
  header: string;
  accent: string;
  mode: 'dark' | 'light';
  dots?: { close: string; min: string; max: string };
  colors?: string[];
}

export const ThemePreview: React.FC = () => {
  const themes: ThemeInfo[] = [
    {
      id: 'tokyo-night',
      name: 'Tokyo Night',
      tag: 'Default Dark',
      bg: '#1a1b26',
      fg: '#c0caf5',
      header: '#13141c',
      accent: '#7aa2f7',
      mode: 'dark',
      dots: { close: '#f7768e', min: '#e0af68', max: '#9ece6a' },
      colors: ['#f7768e', '#e0af68', '#9ece6a', '#7dcfff', '#7aa2f7', '#bb9af7'],
    },
    {
      id: 'catppuccin-mocha',
      name: 'Catppuccin Mocha',
      tag: 'Charcoal Dark',
      bg: '#1e1e2e',
      fg: '#cdd6f4',
      header: '#181825',
      accent: '#89b4fa',
      mode: 'dark',
      dots: { close: '#f38ba8', min: '#f9e2af', max: '#a6e3a1' },
      colors: ['#f38ba8', '#f9e2af', '#a6e3a1', '#94e2d5', '#89b4fa', '#f5c2e7'],
    },
    {
      id: 'minecraft',
      name: 'Minecraft',
      tag: '',
      bg: '#191c19',
      fg: '#e4e7dd',
      header: '#131513',
      accent: '#55ff55',
      mode: 'dark',
      dots: { close: '#ff4747', min: '#ffaa00', max: '#55ff55' },
      colors: ['#ff5555', '#ffff55', '#55ff55', '#55ffff', '#5a78ff', '#e37bfb'],
    },
    {
      id: 'tokyo-day',
      name: 'Tokyo Day',
      tag: 'Crisp Light',
      bg: '#f2f4f8',
      fg: '#343b58',
      header: '#e6e9ef',
      accent: '#2e7de9',
      mode: 'light',
      dots: { close: '#f52a65', min: '#8c6c3e', max: '#587539' },
      colors: ['#f52a65', '#8c6c3e', '#587539', '#007197', '#2e7de9', '#9854f1'],
    },
    {
      id: 'solarized-light',
      name: 'Solarized Light',
      tag: 'Parchment Light',
      bg: '#fdf6e3',
      fg: '#657b83',
      header: '#eee8d5',
      accent: '#268bd2',
      mode: 'light',
      dots: { close: '#dc322f', min: '#b58900', max: '#859900' },
      colors: ['#dc322f', '#b58900', '#859900', '#2aa198', '#268bd2', '#d33682'],
    },
  ];

  const [selectedTheme, setSelectedTheme] = useState<ThemeInfo>(themes[0]);

  return (
    <section id="themes" className="py-20 hairline-b bg-[#13141c]">
      <div className="max-w-7xl mx-auto px-6">
        {/* Section Header */}
        <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-12">
          <div>
            <div className="font-mono text-xs text-[#7aa2f7] uppercase tracking-wider mb-2">
              COLOR THEMES
            </div>
            <h2 className="text-3xl sm:text-5xl font-bold tracking-tight text-white">
              Built-in color palettes
            </h2>
          </div>
          <p className="font-mono text-xs text-[#565f89] max-w-md">
            Full 24-bit TrueColor (16.7 million colors) support. Switch themes with <code>Ctrl+,</code> or configure via <code>config.json</code>.
          </p>
        </div>

        {/* Theme Selector & Live Preview Box */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
          {/* Theme List */}
          <div className="lg:col-span-5 flex flex-col gap-3 font-mono text-xs">
            <div className="text-[#565f89] uppercase tracking-wider mb-1 text-[11px]">
              Select Palette:
            </div>

            {themes.map((th) => (
              <button
                key={th.id}
                onClick={() => setSelectedTheme(th)}
                className={`p-3.5 rounded-[2px] border text-left transition-colors flex items-center justify-between ${
                  selectedTheme.id === th.id
                    ? 'bg-[#1a1b26] border-[#7aa2f7]'
                    : 'bg-[#16161e] border-[#292e42] hover:border-[#565f89]'
                }`}
              >
                <div className="flex items-center gap-3">
                  <div className="w-4 h-4 rounded-full border border-black/20" style={{ backgroundColor: th.bg }} />
                  <div>
                    <div className="font-bold text-white text-sm">{th.name}</div>
                    {th.tag && <div className="text-[11px] text-[#565f89]">{th.tag}</div>}
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  {th.mode === 'dark' ? (
                    <Moon className="w-3.5 h-3.5 text-[#7aa2f7]" />
                  ) : (
                    <Sun className="w-3.5 h-3.5 text-[#e0af68]" />
                  )}
                  {selectedTheme.id === th.id && <Check className="w-4 h-4 text-[#9ece6a]" />}
                </div>
              </button>
            ))}

            {/* FreeDesktop portal note */}
            <div className="p-3.5 bg-[#16161e] border border-[#292e42] rounded-[2px] flex items-start gap-3 text-xs text-[#a9b1d6]">
              <Monitor className="w-4 h-4 text-[#7aa2f7] shrink-0 mt-0.5" />
              <div>
                <span className="font-bold text-white">System Appearance Sync:</span> Automatically follows FreeDesktop DBus dark/light mode preference across GNOME, KDE, and Sway.
              </div>
            </div>
          </div>

          {/* Live Preview Window */}
          <div className="lg:col-span-7">
            <div
              className={`rounded-[4px] overflow-hidden border shadow-2xl transition-colors duration-150 ${
                selectedTheme.id === 'minecraft'
                  ? 'border-2 border-black shadow-[inset_2px_2px_0_#4a4e4a,inset_-2px_-2px_0_#141614]'
                  : 'border-[#292e42]'
              }`}
              style={{ backgroundColor: selectedTheme.bg }}
            >
              {/* Header */}
              <div
                className="h-8 flex items-center justify-between px-3 border-b transition-colors duration-150"
                style={{
                  backgroundColor: selectedTheme.header,
                  borderColor: selectedTheme.mode === 'dark' ? '#292e42' : '#d0d7de',
                }}
              >
                <div className="flex items-center gap-1.5">
                  {selectedTheme.id === 'minecraft' ? (
                    <>
                      <div
                        className="w-3 h-3 bg-[#b81818] border border-black shadow-[inset_1px_1px_0_#ff6666,inset_-1px_-1px_0_#660000]"
                        title="Redstone (Close)"
                      />
                      <div
                        className="w-3 h-3 bg-[#dca316] border border-black shadow-[inset_1px_1px_0_#ffea75,inset_-1px_-1px_0_#7a5700]"
                        title="Gold (Minimize)"
                      />
                      <div
                        className="w-3 h-3 bg-[#1db347] border border-black shadow-[inset_1px_1px_0_#70ff94,inset_-1px_-1px_0_#0c6922]"
                        title="Emerald (Maximize)"
                      />
                    </>
                  ) : (
                    <>
                      <div
                        className="w-2.5 h-2.5 rounded-full"
                        style={{ backgroundColor: selectedTheme.dots?.close || '#f7768e' }}
                      />
                      <div
                        className="w-2.5 h-2.5 rounded-full"
                        style={{ backgroundColor: selectedTheme.dots?.min || '#e0af68' }}
                      />
                      <div
                        className="w-2.5 h-2.5 rounded-full"
                        style={{ backgroundColor: selectedTheme.dots?.max || '#9ece6a' }}
                      />
                    </>
                  )}
                </div>
                <div
                  className={`font-mono text-xs ${selectedTheme.id === 'minecraft' ? '[text-shadow:1px_1px_0px_#153f15] font-bold' : ''}`}
                  style={{ color: selectedTheme.accent }}
                >
                  terrat &mdash; {selectedTheme.name}
                </div>
                <div className="font-mono text-[11px] opacity-50" style={{ color: selectedTheme.fg }}>
                  80x24
                </div>
              </div>

              {/* Terminal Preview Content */}
              <div
                className={`p-5 font-mono text-xs space-y-2 leading-relaxed ${
                  selectedTheme.id === 'minecraft' ? '[text-shadow:1px_1px_0px_#111111]' : ''
                }`}
                style={{ color: selectedTheme.fg }}
              >
                <div className="flex items-center gap-2">
                  <span style={{ color: selectedTheme.accent }} className="font-bold">vixland@box</span>
                  <span className="opacity-50">:</span>
                  <span style={{ color: selectedTheme.accent }}>~/terrat</span>
                  <span className="opacity-50">$</span>
                  <span>git status -s</span>
                  {selectedTheme.id === 'minecraft' && (
                    <span className="inline-block w-2 h-3.5 bg-[#55ffff] border border-black shadow-[inset_1px_1px_0_#aaffff,inset_-1px_-1px_0_#008888]" />
                  )}
                </div>

                <div className="pl-2 space-y-0.5 text-[11px]">
                  <div style={{ color: selectedTheme.colors ? selectedTheme.colors[2] : '#9ece6a' }}>&nbsp;M pkg/render/font.go</div>
                  <div style={{ color: selectedTheme.colors ? selectedTheme.colors[2] : '#9ece6a' }}>&nbsp;M pkg/terminal/tab.go</div>
                  <div style={{ color: selectedTheme.colors ? selectedTheme.colors[0] : '#f7768e' }}>?? pkg/terminal/search.go</div>
                </div>

                <div className="flex items-center gap-2 pt-2">
                  <span style={{ color: selectedTheme.accent }} className="font-bold">vixland@box</span>
                  <span className="opacity-50">:</span>
                  <span style={{ color: selectedTheme.accent }}>~/terrat</span>
                  <span className="opacity-50">$</span>
                  <span>go test -v ./pkg/terminal</span>
                </div>

                <div className="pl-2 space-y-0.5 text-[11px]">
                  <div>=== RUN   TestTabCreation</div>
                  <div style={{ color: selectedTheme.colors ? selectedTheme.colors[2] : '#9ece6a' }}>--- PASS: TestTabCreation (0.00s)</div>
                  <div>=== RUN   TestPTYAllocation</div>
                  <div style={{ color: selectedTheme.colors ? selectedTheme.colors[2] : '#9ece6a' }}>--- PASS: TestPTYAllocation (0.01s)</div>
                  <div>=== RUN   TestSearchBuffer</div>
                  <div style={{ color: selectedTheme.colors ? selectedTheme.colors[2] : '#9ece6a' }}>--- PASS: TestSearchBuffer (0.00s)</div>
                  <div style={{ color: selectedTheme.colors ? selectedTheme.colors[2] : '#9ece6a' }} className="font-bold">PASS</div>
                  <div className="opacity-75">ok&nbsp;&nbsp;github.com/vixland4509/Terrat/pkg/terminal&nbsp;&nbsp;0.012s</div>
                </div>

                {/* ANSI color swatches */}
                <div className="pt-3 border-t border-black/10 flex gap-1">
                  {(selectedTheme.colors || ['#f7768e', '#e0af68', '#9ece6a', '#7dcfff', '#7aa2f7', '#bb9af7']).map((hex) => (
                    <div
                      key={hex}
                      className="w-6 h-3.5 rounded-[1px]"
                      style={{ backgroundColor: hex }}
                      title={hex}
                    />
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
