import React from 'react';
import { GithubIcon } from './GithubIcon';

export const Footer: React.FC = () => {
  return (
    <footer className="py-10 bg-[#13141c] text-[#565f89] font-mono text-xs hairline-t">
      <div className="max-w-7xl mx-auto px-6 flex flex-col sm:flex-row items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <img src="./icon.png" alt="Terrat Logo" className="w-5 h-5 rounded-full border border-[#292e42]" />
          <div>
            <span className="text-[#c0caf5] font-semibold">TerraTerminal</span> &mdash; Pure Go terminal emulator for Linux
          </div>
        </div>

        <div className="flex items-center gap-5 text-[11px]">
          <span>GNU GPLv3 License</span>
          <span className="text-[#292e42]">&bull;</span>
          <a
            href="https://github.com/vixland4509/Terrat"
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-[#c0caf5] transition-colors flex items-center gap-1.5 text-[#7aa2f7]"
          >
            <GithubIcon className="w-3.5 h-3.5" />
            <span>github.com/vixland4509/Terrat</span>
          </a>
        </div>
      </div>
    </footer>
  );
};
