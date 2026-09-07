import React, { useState, useEffect, useRef } from 'react';
import { Terminal as TerminalIcon, Search, ExternalLink, Plus, X, ArrowRight } from 'lucide-react';

interface HistoryItem {
  cmd: string;
  output: React.ReactNode;
}

export const HeroTerminal: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'bash' | 'htop' | 'nvim'>('bash');
  const [inputVal, setInputVal] = useState('');
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [history, setHistory] = useState<HistoryItem[]>([
    {
      cmd: 'terrat --version',
      output: (
        <div className="text-[#a9b1d6] font-mono text-xs">
          terrat 0.1.0 (linux/amd64, pure-go/xgb, no-cgo)<br />
          kernel: linux 6.10 &bull; x11 wire protocol
        </div>
      ),
    },
    {
      cmd: 'ps -o pid,rss,vsz,comm -C terrat',
      output: (
        <div className="font-mono text-xs text-[#a9b1d6]">
          <div className="text-[#565f89]">&nbsp;&nbsp;PID&nbsp;&nbsp;&nbsp;RSS&nbsp;&nbsp;&nbsp;&nbsp;VSZ&nbsp;COMMAND</div>
          <div>&nbsp;4821&nbsp;<span className="text-[#9ece6a] font-bold">11264</span>&nbsp;712340&nbsp;terrat</div>
        </div>
      ),
    },
  ]);

  const terminalBodyRef = useRef<HTMLDivElement>(null);

  const handleRunCommand = (cmd: string) => {
    const trimmed = cmd.trim();
    if (!trimmed) return;

    let out: React.ReactNode = null;
    const lower = trimmed.toLowerCase();

    if (lower === 'clear') {
      setHistory([]);
      setInputVal('');
      return;
    } else if (lower.includes('version') || lower === 'terrat -v') {
      out = (
        <div className="text-[#a9b1d6] font-mono text-xs">
          terrat 0.1.0 (linux/amd64, pure-go/xgb, no-cgo)<br />
          kernel: linux 6.10 &bull; x11 wire protocol
        </div>
      );
    } else if (lower.includes('ps') || lower.includes('mem')) {
      out = (
        <div className="font-mono text-xs text-[#a9b1d6]">
          <div className="text-[#565f89]">&nbsp;&nbsp;PID&nbsp;&nbsp;&nbsp;RSS&nbsp;&nbsp;&nbsp;&nbsp;VSZ&nbsp;COMMAND</div>
          <div>&nbsp;4821&nbsp;<span className="text-[#9ece6a] font-bold">11264</span>&nbsp;712340&nbsp;terrat</div>
        </div>
      );
    } else if (lower.includes('uname')) {
      out = (
        <div className="text-xs text-[#a9b1d6]">
          Linux 6.10.9-arch1-1 x86_64 GNU/Linux
        </div>
      );
    } else if (lower.includes('colors')) {
      out = (
        <div className="space-y-1.5 pt-1 text-xs">
          <div className="flex gap-1">
            {['#15161e', '#f7768e', '#9ece6a', '#e0af68', '#7aa2f7', '#bb9af7', '#7dcfff', '#a9b1d6'].map((c, i) => (
              <span key={i} className="px-2 py-0.5 text-[10px] text-black font-bold" style={{ backgroundColor: c }}>
                {i}
              </span>
            ))}
          </div>
          <div className="flex gap-1">
            {['#414868', '#ff9e64', '#73daca', '#b4f9f8', '#2ac3de', '#c0caf5', '#e0af68', '#cfc9c2'].map((c, i) => (
              <span key={i} className="px-2 py-0.5 text-[10px] text-black font-bold" style={{ backgroundColor: c }}>
                {i + 8}
              </span>
            ))}
          </div>
        </div>
      );
    } else if (lower === 'terrat --help' || lower === 'help') {
      out = (
        <div className="text-xs text-[#c0caf5] space-y-2 font-mono">
          <div>Usage: terrat [flags]</div>
          <div className="text-[#565f89]">Flags:</div>
          <div className="pl-4 space-y-0.5">
            <div><span className="text-[#7aa2f7]">-config</span> string&nbsp;&nbsp;&nbsp;&nbsp;Path to JSON config (default ~/.config/terrat/config.json)</div>
            <div><span className="text-[#7aa2f7]">-theme</span> string&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Color theme (tokyo-night, catppuccin-mocha, tokyo-day, solarized-light)</div>
            <div><span className="text-[#7aa2f7]">-font-size</span> float&nbsp;Base font size in points (default 12.0)</div>
            <div><span className="text-[#7aa2f7]">-opacity</span> float&nbsp;&nbsp;&nbsp;Window transparency alpha [0.2 - 1.0] (default 0.95)</div>
            <div><span className="text-[#7aa2f7]">-v, --version</span>&nbsp;&nbsp;&nbsp;&nbsp;Print version and exit</div>
          </div>
          <div className="text-[#565f89] pt-1">Default Keybindings:</div>
          <div className="pl-4 text-[11px] text-[#565f89] space-y-0.5">
            <div>Ctrl+Shift+T: new tab &bull; Ctrl+Tab: cycle tab &bull; Ctrl+Shift+F: find</div>
            <div>Ctrl+= / Ctrl+-: zoom font &bull; Ctrl+Click: launch URL &bull; Ctrl+,: preferences</div>
          </div>
        </div>
      );
    } else {
      out = (
        <div className="text-xs text-[#f7768e]">
          bash: {trimmed}: command not found. Try running <span className="text-[#7aa2f7] underline cursor-pointer" onClick={() => handleRunCommand('terrat --help')}>terrat --help</span>, <span className="text-[#7aa2f7] underline cursor-pointer" onClick={() => handleRunCommand('colors')}>colors</span>, or <span className="text-[#7aa2f7] underline cursor-pointer" onClick={() => handleRunCommand('ps')}>ps</span>.
        </div>
      );
    }

    setHistory((prev) => [...prev, { cmd: trimmed, output: out }]);
    setInputVal('');
    setTimeout(() => {
      if (terminalBodyRef.current) {
        terminalBodyRef.current.scrollTop = terminalBodyRef.current.scrollHeight;
      }
    }, 20);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleRunCommand(inputVal);
    }
  };

  return (
    <section className="relative pt-16 pb-20 overflow-hidden hairline-b bg-grid-pattern">
      <div className="max-w-7xl mx-auto px-6">

        {/* Hero Title & Grounded Description */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 items-end mb-16">
          <div className="lg:col-span-8">
            <h1 className="text-4xl sm:text-6xl lg:text-7xl font-bold tracking-tight text-white leading-[1.05] mb-6">
              A lightweight Linux terminal, <br />
              <span className="text-[#7aa2f7]">written in pure Go.</span>
            </h1>
            <p className="text-base sm:text-lg text-[#a9b1d6] max-w-2xl leading-relaxed">
              No CGO bindings, no Electron, and no heavy GPU framework requirements.
              Terrat talks directly to the X11 display server via native wire protocol over Unix domain sockets.
              Cold boots in under 5ms, stays at ~11MB of memory, and includes multi-tabs, live font scaling, and buffer search out of the box.
            </p>
          </div>

          <div className="lg:col-span-4 flex flex-col gap-4 font-mono text-xs">
            <div className="p-4 bg-[#16161e] border border-[#292e42] rounded-[2px]">
              <div className="text-[#565f89] mb-1.5 uppercase">QUICK INSTALL</div>
              <div className="text-[#c0caf5] font-semibold break-all selection:bg-[#7aa2f7] selection:text-black">
                go install github.com/vixland4509/Terrat@latest
              </div>
            </div>
            <div className="flex items-center gap-3">
              <a
                href="#install"
                className="flex-1 text-center py-2.5 bg-[#7aa2f7] hover:bg-[#89b4fa] text-[#13141c] font-bold rounded-[2px] transition-all uppercase tracking-wider text-xs"
              >
                Installation &rarr;
              </a>
              <a
                href="https://github.com/vixland4509/Terrat"
                target="_blank"
                rel="noopener noreferrer"
                className="flex-1 text-center py-2.5 bg-[#1a1b26] hover:bg-[#24283b] border border-[#292e42] text-[#c0caf5] font-semibold rounded-[2px] transition-all uppercase tracking-wider text-xs"
              >
                Source Code
              </a>
            </div>
          </div>
        </div>

        {/* Interactive Live Terminal Window (Replica of Terrat) */}
        <div className="relative max-w-5xl mx-auto shadow-2xl rounded-[4px] overflow-hidden border border-[#292e42] bg-[#1a1b26]">
          {/* Terrat Custom Headerbar (CSD) */}
          <div className="h-9 bg-[#13141c] border-b border-[#292e42] flex items-center justify-between px-3 select-none">
            {/* macOS / Ghostty Window Dots */}
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-[#f7768e]/80 hover:bg-[#f7768e] transition-colors cursor-pointer" title="Close Window" />
              <div className="w-3 h-3 rounded-full bg-[#e0af68]/80 hover:bg-[#e0af68] transition-colors cursor-pointer" title="Minimize" />
              <div className="w-3 h-3 rounded-full bg-[#9ece6a]/80 hover:bg-[#9ece6a] transition-colors cursor-pointer" title="Maximize" />
            </div>

            {/* Interactive Tabs */}
            <div className="flex items-center gap-1 font-mono text-xs overflow-x-auto">
              <button
                onClick={() => setActiveTab('bash')}
                className={`flex items-center gap-2 px-3 py-1 border-r border-[#292e42] transition-colors ${
                  activeTab === 'bash' ? 'bg-[#1a1b26] text-[#c0caf5]' : 'text-[#565f89] hover:text-[#c0caf5]'
                }`}
              >
                <span>1: bash</span>
                <X className="w-3 h-3 opacity-60 hover:opacity-100" />
              </button>

              <button
                onClick={() => setActiveTab('htop')}
                className={`flex items-center gap-2 px-3 py-1 border-r border-[#292e42] transition-colors ${
                  activeTab === 'htop' ? 'bg-[#1a1b26] text-[#c0caf5]' : 'text-[#565f89] hover:text-[#c0caf5]'
                }`}
              >
                <span>2: htop</span>
                <X className="w-3 h-3 opacity-60 hover:opacity-100" />
              </button>

              <button
                onClick={() => setActiveTab('nvim')}
                className={`flex items-center gap-2 px-3 py-1 border-r border-[#292e42] transition-colors ${
                  activeTab === 'nvim' ? 'bg-[#1a1b26] text-[#c0caf5]' : 'text-[#565f89] hover:text-[#c0caf5]'
                }`}
              >
                <span>3: main.go</span>
                <X className="w-3 h-3 opacity-60 hover:opacity-100" />
              </button>

              <button
                onClick={() => handleRunCommand('features')}
                className="flex items-center justify-center w-6 h-5 rounded-[2px] bg-[#1a1b26] hover:bg-[#24283b] border border-[#292e42] hover:border-[#7aa2f7] text-[#7aa2f7] hover:text-white transition-all ml-1.5 shrink-0"
                title="Open New Tab"
              >
                <Plus className="w-3.5 h-3.5" />
              </button>
            </div>

            {/* Micro-HUD */}
            <div className="flex items-center gap-3 font-mono text-[11px] text-[#565f89]">
              <button
                onClick={() => setIsSearchOpen(!isSearchOpen)}
                className={`hover:text-[#c0caf5] transition-colors flex items-center gap-1 ${
                  isSearchOpen ? 'text-[#7aa2f7]' : ''
                }`}
                title="Toggle In-Buffer Search (Ctrl+Shift+F)"
              >
                <Search className="w-3 h-3" />
                <span className="hidden sm:inline">FIND</span>
              </button>
              <span className="hidden sm:inline border-l border-[#292e42] pl-3 text-[#7aa2f7]">80x24</span>
            </div>
          </div>

          {/* Floating Search Bar Mockup (when open) */}
          {isSearchOpen && (
            <div className="absolute top-11 right-4 z-20 w-72 bg-[#13141c] border border-[#292e42] rounded-[2px] p-2 flex items-center gap-2 font-mono text-xs shadow-xl animate-in fade-in duration-150">
              <span className="text-[#7aa2f7] font-bold">FIND:</span>
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search buffer..."
                className="bg-transparent border-none outline-none text-[#c0caf5] flex-1 text-xs font-mono placeholder:text-[#565f89]"
                autoFocus
              />
              <span className="text-[#565f89] text-[10px]">{searchQuery ? '(1/3)' : '(0/0)'}</span>
              <button onClick={() => setIsSearchOpen(false)} className="text-[#565f89] hover:text-[#c0caf5]">
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          )}

          {/* Terminal Content Area */}
          <div ref={terminalBodyRef} className="p-6 font-mono text-sm min-h-[360px] max-h-[460px] overflow-y-auto">
            {activeTab === 'bash' && (
              <div className="space-y-3">
                <div className="text-xs text-[#565f89] border-b border-[#24283b] pb-2 mb-3">
                  Click a command or type directly in the prompt below:
                </div>

                {/* Interactive command chips */}
                <div className="flex flex-wrap gap-2 mb-4">
                  {['terrat --help', 'ps', 'colors', 'clear'].map((cmd) => (
                    <button
                      key={cmd}
                      onClick={() => handleRunCommand(cmd)}
                      className="px-2.5 py-1 text-xs bg-[#16161e] hover:bg-[#24283b] border border-[#292e42] text-[#7aa2f7] rounded-[2px] transition-colors"
                    >
                      $ {cmd}
                    </button>
                  ))}
                </div>

                {/* History */}
                {history.map((item, idx) => (
                  <div key={idx} className="space-y-1">
                    <div className="flex items-center gap-2 text-[#c0caf5]">
                      <span className="text-[#9ece6a] font-bold">vixland@box</span>
                      <span className="text-[#565f89]">:</span>
                      <span className="text-[#7aa2f7]">~</span>
                      <span className="text-[#565f89]">$</span>
                      <span>{item.cmd}</span>
                    </div>
                    <div className="pl-4">{item.output}</div>
                  </div>
                ))}

                {/* Live Interactive Prompt */}
                <div className="flex items-center gap-2 text-[#c0caf5] pt-1">
                  <span className="text-[#9ece6a] font-bold">vixland@box</span>
                  <span className="text-[#565f89]">:</span>
                  <span className="text-[#7aa2f7]">~</span>
                  <span className="text-[#565f89]">$</span>
                  <input
                    type="text"
                    value={inputVal}
                    onChange={(e) => setInputVal(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder="terrat --help"
                    className="bg-transparent border-none outline-none text-[#c0caf5] flex-1 font-mono text-sm placeholder:text-[#565f89]/50"
                  />
                </div>

                {/* URL Hover Showcase */}
                <div className="pt-4 text-xs text-[#565f89] flex items-center gap-2">
                  <span>Repository:</span>
                  <a
                    href="https://github.com/vixland4509/Terrat"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-[#7aa2f7] hover:underline underline-offset-4 flex items-center gap-1"
                  >
                    <span>https://github.com/vixland4509/Terrat</span>
                    <ExternalLink className="w-3 h-3" />
                  </a>
                  <span className="text-[10px] text-[#565f89]">(Ctrl+Click to open)</span>
                </div>
              </div>
            )}

            {activeTab === 'htop' && (
              <div className="space-y-2 text-xs font-mono text-[#c0caf5]">
                <div className="text-[#565f89] space-y-0.5">
                  <div>top - 23:14:02 up 4 days, 12:44,  1 user,  load average: 0.12, 0.08, 0.03</div>
                  <div>Tasks: <span className="text-white">214</span> total,   <span className="text-[#9ece6a]">1</span> running, <span className="text-white">213</span> sleeping</div>
                  <div>%Cpu(s):  <span className="text-white">0.4</span> us,  <span className="text-white">0.2</span> sy,  <span className="text-white">0.0</span> ni, <span className="text-[#9ece6a]">99.4</span> id,  <span className="text-white">0.0</span> wa</div>
                  <div>MiB Mem :  <span className="text-white">31824.2</span> total, <span className="text-white">18420.1</span> free,   <span className="text-[#e0af68]">6124.5</span> used,   <span className="text-white">7279.6</span> buff/cache</div>
                </div>

                <div className="mt-3 pt-2 border-t border-[#292e42]">
                  <table className="w-full text-left">
                    <thead>
                      <tr className="text-[#565f89] uppercase text-[10px]">
                        <th>PID</th>
                        <th>USER</th>
                        <th>PR</th>
                        <th>VIRT</th>
                        <th>RES</th>
                        <th>S</th>
                        <th>%CPU</th>
                        <th>%MEM</th>
                        <th>COMMAND</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-[#24283b]">
                      <tr className="text-[#9ece6a]">
                        <td>4821</td>
                        <td>vixland</td>
                        <td>20</td>
                        <td>712M</td>
                        <td className="font-bold">11.0M</td>
                        <td>S</td>
                        <td>0.0</td>
                        <td>0.03</td>
                        <td>terrat</td>
                      </tr>
                      <tr className="text-[#c0caf5]">
                        <td>4822</td>
                        <td>vixland</td>
                        <td>20</td>
                        <td>14M</td>
                        <td>4.1M</td>
                        <td>S</td>
                        <td>0.0</td>
                        <td>0.01</td>
                        <td>bash</td>
                      </tr>
                      <tr className="text-[#565f89]">
                        <td>1204</td>
                        <td>vixland</td>
                        <td>20</td>
                        <td>842M</td>
                        <td>64.2M</td>
                        <td>S</td>
                        <td>0.8</td>
                        <td>0.20</td>
                        <td>picom --backend xrender</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {activeTab === 'nvim' && (
              <div className="text-xs space-y-1 font-mono text-[#c0caf5]">
                <div className="text-[#565f89] mb-2">cmd/terrat/main.go</div>
                <div><span className="text-[#f7768e]">package</span> main</div>
                <div className="pt-2"><span className="text-[#f7768e]">import</span> (</div>
                <div className="pl-4 text-[#9ece6a]">"os"</div>
                <div className="pl-4 text-[#9ece6a]">"github.com/vixland4509/Terrat/pkg/config"</div>
                <div className="pl-4 text-[#9ece6a]">"github.com/vixland4509/Terrat/pkg/platform"</div>
                <div className="pl-4 text-[#9ece6a]">"github.com/vixland4509/Terrat/pkg/terminal"</div>
                <div>)</div>
                <div className="pt-2"><span className="text-[#f7768e]">func</span> <span className="text-[#7aa2f7]">main</span>() &#123;</div>
                <div className="pl-4 text-[#c0caf5]">
                  cfg := config.<span className="text-[#7dcfff]">LoadOrCreate</span>()
                </div>
                <div className="pl-4 text-[#c0caf5]">
                  win, err := platform.<span className="text-[#7dcfff]">NewWindow</span>(<span className="text-[#9ece6a]">"TerraTerminal"</span>, cfg)
                </div>
                <div className="pl-4 text-[#565f89]">if err != nil &#123; os.Exit(1) &#125;</div>
                <div className="pl-4 text-[#c0caf5]">
                  defer win.<span className="text-[#7dcfff]">Close</span>()
                </div>
                <div className="pl-4 text-[#c0caf5]">
                  app := terminal.<span className="text-[#7dcfff]">NewApp</span>(win, cfg)
                </div>
                <div className="pl-4 text-[#c0caf5]">
                  app.<span className="text-[#7dcfff]">Run</span>()
                </div>
                <div>&#125;</div>
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  );
};
