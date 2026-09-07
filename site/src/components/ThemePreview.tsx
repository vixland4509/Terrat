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
                    <div className="text-[11px] text-[#565f89]">{th.tag}</div>
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
              className="rounded-[4px] overflow-hidden border border-[#292e42] shadow-2xl transition-colors duration-150"
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
                  <div className="w-2.5 h-2.5 rounded-full bg-[#f7768e]" />
                  <div className="w-2.5 h-2.5 rounded-full bg-[#e0af68]" />
                  <div className="w-2.5 h-2.5 rounded-full bg-[#9ece6a]" />
                </div>
                <div className="font-mono text-xs" style={{ color: selectedTheme.accent }}>
                  terrat &mdash; {selectedTheme.name}
                </div>
                <div className="font-mono text-[11px] opacity-50" style={{ color: selectedTheme.fg }}>
                  80x24
                </div>
              </div>

              {/* Terminal Preview Content */}
              <div className="p-5 font-mono text-xs space-y-2 leading-relaxed" style={{ color: selectedTheme.fg }}>
                <div className="flex items-center gap-2">
                  <span style={{ color: selectedTheme.accent }} className="font-bold">vixland@box</span>
                  <span className="opacity-50">:</span>
                  <span style={{ color: selectedTheme.accent }}>~/terrat</span>
                  <span className="opacity-50">$</span>
                  <span>git status -s</span>
                </div>

                <div className="pl-2 space-y-0.5 text-[11px]">
                  <div className="text-[#9ece6a]">&nbsp;M pkg/render/font.go</div>
                  <div className="text-[#9ece6a]">&nbsp;M pkg/terminal/tab.go</div>
                  <div className="text-[#f7768e]">?? pkg/terminal/search.go</div>
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
                  <div className="text-[#9ece6a]">--- PASS: TestTabCreation (0.00s)</div>
                  <div>=== RUN   TestPTYAllocation</div>
                  <div className="text-[#9ece6a]">--- PASS: TestPTYAllocation (0.01s)</div>
                  <div>=== RUN   TestSearchBuffer</div>
                  <div className="text-[#9ece6a]">--- PASS: TestSearchBuffer (0.00s)</div>
                  <div className="text-[#9ece6a] font-bold">PASS</div>
                  <div className="opacity-75">ok&nbsp;&nbsp;github.com/vixland4509/Terrat/pkg/terminal&nbsp;&nbsp;0.012s</div>
                </div>

                {/* ANSI color swatches */}
                <div className="pt-3 border-t border-black/10 flex gap-1">
                  {['#f7768e', '#e0af68', '#9ece6a', '#7dcfff', '#7aa2f7', '#bb9af7'].map((hex) => (
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
