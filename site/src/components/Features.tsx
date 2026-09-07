import React from 'react';
import { Layers, ZoomIn, Search, Link2, Eye, MousePointer } from 'lucide-react';

export const Features: React.FC = () => {
  const featureList = [
    {
      num: '01',
      title: 'Multi-Tabs (Isolated PTY)',
      subtitle: 'POSIX Pseudo-Terminal Sessions',
      desc: 'Each tab allocates an independent PTY master/slave pair (/dev/ptmx). Shell processes are fully isolated. Exiting a shell automatically closes its tab, and closing the final tab cleanly exits the application.',
      shortcut: 'Ctrl+Shift+T / Ctrl+Tab / Alt+1..9',
      icon: Layers,
      color: '#7aa2f7',
    },
    {
      num: '02',
      title: 'Live Font Scaling',
      subtitle: 'Dynamic Rasterization & SIGWINCH',
      desc: 'Adjust font size dynamically without restarting. TrueType vector glyphs are rendered into memory alpha masks in sub-millisecond time. The window geometry reflows and transmits SIGWINCH to active processes.',
      shortcut: 'Ctrl+= / Ctrl+- / Ctrl+0 / Ctrl+Wheel',
      icon: ZoomIn,
      color: '#9ece6a',
    },
    {
      num: '03',
      title: 'In-Buffer Search',
      subtitle: 'Live Scrollback Matching',
      desc: 'Floating find bar searches through active screen rows and scrollback history. Highlights all matches with distinct active match indicators. Navigate forward and backward with Enter and Shift+Enter.',
      shortcut: 'Ctrl+Shift+F / Enter / Shift+Enter',
      icon: Search,
      color: '#e0af68',
    },
    {
      num: '04',
      title: 'URL Detection & Launch',
      subtitle: 'Async xdg-open Integration',
      desc: 'Scans visible cells for web URLs and file paths. Holding Ctrl highlights the detected URL and changes the X11 mouse cursor to a pointer. Ctrl+Click opens the URL in your default browser asynchronously.',
      shortcut: 'Ctrl + Hover / Ctrl + Left Click',
      icon: Link2,
      color: '#7dcfff',
    },
    {
      num: '05',
      title: 'Composited Window Opacity',
      subtitle: 'EWMH _NET_WM_WINDOW_OPACITY',
      desc: 'Communicates directly with your X11 window manager and compositor (Picom, Mutter, KWin). Transparency is handled by the compositor, keeping terminal rendering code fast and simple.',
      shortcut: 'Configured in ~/.config/terrat/config.json',
      icon: Eye,
      color: '#bb9af7',
    },
    {
      num: '06',
      title: 'Dual Selection & Clipboard',
      subtitle: 'PRIMARY & CLIPBOARD Sync',
      desc: 'Click-drag character selection with double-click word/path selection and triple-click line selection. Automatically updates X11 PRIMARY for middle-click paste and CLIPBOARD for Ctrl+Shift+V.',
      shortcut: 'Double-Click / Triple-Click / Middle-Click',
      icon: MousePointer,
      color: '#f7768e',
    },
  ];

  return (
    <section id="features" className="py-20 hairline-b bg-[#1a1b26]">
      <div className="max-w-7xl mx-auto px-6">
        {/* Section Header */}
        <div className="mb-14">
          <div className="font-mono text-xs text-[#7aa2f7] uppercase tracking-wider mb-2">
            CORE FEATURES
          </div>
          <h2 className="text-3xl sm:text-5xl font-bold tracking-tight text-white mb-3">
            Practical features for everyday terminal work.
          </h2>
          <p className="text-sm sm:text-base text-[#a9b1d6] max-w-2xl">
            Built-in capabilities designed for Linux workstations without requiring external multiplexers or heavy desktop frameworks.
          </p>
        </div>

        {/* Feature Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {featureList.map((f, i) => {
            const Icon = f.icon;
            return (
              <div
                key={i}
                className="p-6 bg-[#16161e] border border-[#292e42] hover:border-[#3b4261] rounded-[2px] transition-colors flex flex-col justify-between"
              >
                <div>
                  <div className="flex items-center justify-between mb-4">
                    <span className="font-mono text-xs text-[#565f89]">
                      {f.num}
                    </span>
                    <Icon className="w-5 h-5" style={{ color: f.color }} />
                  </div>

                  <h3 className="text-lg font-bold text-white mb-1">{f.title}</h3>
                  <div className="font-mono text-[11px] text-[#7aa2f7] mb-3">{f.subtitle}</div>
                  <p className="text-xs text-[#a9b1d6] leading-relaxed mb-6">
                    {f.desc}
                  </p>
                </div>

                <div className="pt-3 border-t border-[#292e42] font-mono text-[11px] text-[#565f89]">
                  Shortcut: <span className="text-[#c0caf5]">{f.shortcut}</span>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
};
