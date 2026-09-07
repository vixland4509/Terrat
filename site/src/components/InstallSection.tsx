import React, { useState } from 'react';
import { Copy, Check, Terminal, Download, Keyboard, Settings } from 'lucide-react';

export const InstallSection: React.FC = () => {
  const [copiedTab, setCopiedTab] = useState<string | null>(null);

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedTab(id);
    setTimeout(() => setCopiedTab(null), 2000);
  };

  const keybindings = [
    { key: 'Ctrl + Shift + T', desc: 'Spawn new tab' },
    { key: 'Ctrl + Shift + W', desc: 'Close active tab' },
    { key: 'Ctrl + Tab', desc: 'Cycle to next tab' },
    { key: 'Alt + 1..9', desc: 'Jump to tab 1..9' },
    { key: 'Ctrl + Shift + F', desc: 'Toggle in-buffer search' },
    { key: 'Ctrl + = / - / 0', desc: 'Zoom font size in / out / reset' },
    { key: 'Ctrl + Click', desc: 'Open URL in default browser' },
    { key: 'Ctrl + ,', desc: 'Open preferences & themes modal' },
    { key: 'Ctrl + Shift + C', desc: 'Copy selected buffer text' },
    { key: 'Ctrl + Shift + V', desc: 'Paste text from clipboard' },
    { key: 'Shift + PageUp / Down', desc: 'Scroll buffer half-page' },
    { key: 'Mouse Middle Click', desc: 'Paste X11 PRIMARY selection' },
  ];

  return (
    <section id="install" className="py-20 hairline-b bg-[#1a1b26]">
      <div className="max-w-7xl mx-auto px-6">
        {/* Section Header */}
        <div className="mb-12">
          <div className="font-mono text-xs text-[#7aa2f7] uppercase tracking-wider mb-2">
            INSTALLATION &amp; CONFIGURATION
          </div>
          <h2 className="text-3xl sm:text-5xl font-bold tracking-tight text-white mb-3">
            Install and run in seconds.
          </h2>
          <p className="text-sm sm:text-base text-[#a9b1d6] max-w-2xl">
            Because Terrat is written in pure Go without CGO, building produces a single static binary without dependency friction.
          </p>
        </div>

        {/* Installation Boxes */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-12">
          {/* Method 1: Go Install */}
          <div className="p-6 bg-[#16161e] border border-[#292e42] rounded-[2px] flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between mb-3">
                <span className="font-mono text-xs text-[#7aa2f7] uppercase tracking-wider font-bold">
                  Method 1: Go Install
                </span>
                <Terminal className="w-4 h-4 text-[#7aa2f7]" />
              </div>
              <p className="text-xs text-[#a9b1d6] mb-4">
                Installs binary directly to your <code>$GOPATH/bin</code> (requires Go 1.21+):
              </p>
              <div className="p-3.5 bg-[#13141c] border border-[#292e42] rounded-[2px] font-mono text-xs flex items-center justify-between gap-4">
                <code className="text-[#c0caf5] select-all">go install github.com/vixland4509/Terrat@latest</code>
                <button
                  onClick={() => copyToClipboard('go install github.com/vixland4509/Terrat@latest', 'go')}
                  className="text-[#565f89] hover:text-[#c0caf5] transition-colors shrink-0"
                  title="Copy command"
                >
                  {copiedTab === 'go' ? <Check className="w-4 h-4 text-[#9ece6a]" /> : <Copy className="w-4 h-4" />}
                </button>
              </div>
            </div>
            <div className="mt-4 text-[11px] font-mono text-[#565f89]">
              Ensure <code>~/go/bin</code> is in your <code>$PATH</code>.
            </div>
          </div>

          {/* Method 2: Build from Source */}
          <div className="p-6 bg-[#16161e] border border-[#292e42] rounded-[2px] flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between mb-3">
                <span className="font-mono text-xs text-[#9ece6a] uppercase tracking-wider font-bold">
                  Method 2: Git &amp; Make
                </span>
                <Download className="w-4 h-4 text-[#9ece6a]" />
              </div>
              <p className="text-xs text-[#a9b1d6] mb-4">
                Clones source, compiles binary, installs icon and desktop launcher:
              </p>
              <div className="p-3.5 bg-[#13141c] border border-[#292e42] rounded-[2px] font-mono text-xs flex items-center justify-between gap-4">
                <pre className="text-[#c0caf5] select-all leading-relaxed whitespace-pre font-mono">
git clone https://github.com/vixland4509/Terrat.git
cd Terrat &amp;&amp; make install
                </pre>
                <button
                  onClick={() =>
                    copyToClipboard(
                      'git clone https://github.com/vixland4509/Terrat.git && cd Terrat && make install',
                      'make'
                    )
                  }
                  className="text-[#565f89] hover:text-[#c0caf5] transition-colors shrink-0 self-start"
                  title="Copy commands"
                >
                  {copiedTab === 'make' ? <Check className="w-4 h-4 text-[#9ece6a]" /> : <Copy className="w-4 h-4" />}
                </button>
              </div>
            </div>
            <div className="mt-4 text-[11px] font-mono text-[#565f89]">
              Installs to <code>~/.local/bin/terrat</code> and registers in application menu.
            </div>
          </div>
        </div>

        {/* Configuration snippet */}
        <div className="border border-[#292e42] bg-[#16161e] rounded-[2px] p-6 mb-8">
          <div className="flex items-center gap-2 font-mono text-xs text-[#7aa2f7] uppercase tracking-wider mb-4">
            <Settings className="w-4 h-4" />
            <span>Configuration (~/.config/terrat/config.json)</span>
          </div>
          <div className="p-4 bg-[#13141c] border border-[#292e42] rounded-[2px] font-mono text-xs text-[#a9b1d6] overflow-x-auto">
            <pre>{`{
  "font": "JetBrains Mono",
  "fontSize": 12.0,
  "theme": "tokyo-night",
  "opacity": 0.95,
  "scrollbackLines": 10000,
  "cursorBlink": true
}`}</pre>
          </div>
          <div className="mt-3 text-[11px] text-[#565f89] font-mono">
            Automatically generated on first launch if not present. All settings can be updated at runtime.
          </div>
        </div>

        {/* Keybindings Table */}
        <div className="border border-[#292e42] bg-[#16161e] rounded-[2px] p-6">
          <div className="font-mono text-xs text-[#7aa2f7] uppercase tracking-wider mb-4 flex items-center gap-2">
            <Keyboard className="w-4 h-4" />
            <span>Default Keyboard Shortcuts</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 font-mono text-xs">
            {keybindings.map((item, idx) => (
              <div
                key={idx}
                className="p-2.5 bg-[#13141c] border border-[#292e42] rounded-[2px] flex items-center justify-between gap-2"
              >
                <span className="text-[#e0af68] font-medium">{item.key}</span>
                <span className="text-[#565f89] text-[11px] text-right">{item.desc}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
};
