import React from 'react';
import { Info } from 'lucide-react';

export const Benchmarks: React.FC = () => {
  const data = [
    {
      metric: 'Cold Startup Time',
      desc: 'Time from execve to initial X11 Expose event and first render',
      terrat: '4.6 ms',
      terratHighlight: true,
      alacritty: '25 ms',
      kitty: '48 ms',
      konsole: '120 ms',
    },
    {
      metric: 'Resident Memory (RSS)',
      desc: 'Physical memory allocated in RAM after launching shell',
      terrat: '11.0 MB',
      terratHighlight: true,
      alacritty: '42.0 MB',
      kitty: '58.0 MB',
      konsole: '168.0 MB',
    },
    {
      metric: 'Proportional Set Size (PSS)',
      desc: 'Unique memory plus proportional share of common libraries',
      terrat: '9.2 MB',
      terratHighlight: true,
      alacritty: '34.5 MB',
      kitty: '47.0 MB',
      konsole: '124.0 MB',
    },
    {
      metric: 'Binary Size',
      desc: 'Standalone executable size (stripped)',
      terrat: '3.7 MB (static)',
      terratHighlight: true,
      alacritty: '14.2 MB',
      kitty: '22.0 MB',
      konsole: 'Multi-MB (Qt/KDE)',
    },
    {
      metric: 'Runtime Dependencies',
      desc: 'Dynamic libraries and graphics stacks required',
      terrat: 'None (Pure Go)',
      terratHighlight: true,
      alacritty: 'Rust + OpenGL / fontconfig',
      kitty: 'Python + C + OpenGL / harfbuzz',
      konsole: 'Qt6 + KDE Frameworks',
    },
    {
      metric: 'Idle CPU Usage',
      desc: 'CPU load when waiting for user input',
      terrat: '0.00% (epoll sleep)',
      terratHighlight: true,
      alacritty: '0.00%',
      kitty: '0.10%',
      konsole: '0.50% - 1.20%',
    },
  ];

  return (
    <section id="benchmarks" className="py-20 hairline-b bg-[#13141c]">
      <div className="max-w-7xl mx-auto px-6">
        {/* Section Header */}
        <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-12">
          <div>
            <div className="font-mono text-xs text-[#7aa2f7] uppercase tracking-wider mb-2">
              RESOURCE FOOTPRINT
            </div>
            <h2 className="text-3xl sm:text-5xl font-bold tracking-tight text-white">
              Performance &amp; Memory Profile
            </h2>
          </div>
          <p className="font-mono text-xs text-[#565f89] max-w-md">
            Tested on Linux 6.10 x86_64 under an X11 session with Picom compositor.
            Measured using <code>perf</code>, <code>time</code>, and <code>/proc/$PID/smaps_rollup</code>.
          </p>
        </div>

        {/* Data-Sheet Table */}
        <div className="border border-[#292e42] rounded-[2px] overflow-x-auto bg-[#16161e] mb-8">
          <table className="w-full text-left font-mono text-xs border-collapse">
            <thead>
              <tr className="bg-[#1a1b26] text-[#565f89] border-b border-[#292e42] uppercase text-[11px] tracking-wider">
                <th className="p-4 border-r border-[#292e42]">METRIC &amp; SPECIFICATION</th>
                <th className="p-4 border-r border-[#292e42] text-[#9ece6a] bg-[#7aa2f7]/5">
                  TERRAT (PURE GO)
                </th>
                <th className="p-4 border-r border-[#292e42]">ALACRITTY</th>
                <th className="p-4 border-r border-[#292e42]">KITTY</th>
                <th className="p-4">KONSOLE</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#292e42]">
              {data.map((row, i) => (
                <tr key={i} className="hover:bg-[#1a1b26]/50 transition-colors">
                  <td className="p-4 border-r border-[#292e42]">
                    <div className="font-semibold text-white">{row.metric}</div>
                    <div className="text-[11px] text-[#565f89] mt-0.5">{row.desc}</div>
                  </td>
                  <td className="p-4 border-r border-[#292e42] font-bold text-[#9ece6a] bg-[#7aa2f7]/5 text-sm">
                    {row.terrat}
                  </td>
                  <td className="p-4 border-r border-[#292e42] text-[#c0caf5]">{row.alacritty}</td>
                  <td className="p-4 border-r border-[#292e42] text-[#c0caf5]">{row.kitty}</td>
                  <td className="p-4 text-[#565f89]">{row.konsole}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Technical Notes & Trade-Offs */}
        <div className="p-5 border border-[#292e42] bg-[#16161e] rounded-[2px] flex items-start gap-3 text-xs text-[#a9b1d6]">
          <Info className="w-5 h-5 text-[#7aa2f7] shrink-0 mt-0.5" />
          <div className="space-y-1">
            <div className="font-bold text-white">Methodology &amp; Engineering Trade-Offs</div>
            <p className="leading-relaxed">
              Terrat achieves sub-5ms cold startup because it bypasses OpenGL/Vulkan context initialization and fontconfig library scans, connecting directly to the X11 socket.
              It is optimized for code editing, command-line tools, and low-latency typing. If you need complex script shaping (HarfBuzz for Arabic or Devanagari) or sixel graphics rendering, Kitty and Alacritty remain excellent choices.
            </p>
          </div>
        </div>
      </div>
    </section>
  );
};
