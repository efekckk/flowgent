import type { SVGProps } from 'react';

type Glyph =
  | 'bolt' | 'chip' | 'signal' | 'cylinder' | 'branch' | 'braces'
  | 'webhook' | 'clock' | 'scroll' | 'key' | 'search' | 'logout'
  | 'plus' | 'play' | 'stop' | 'check' | 'x' | 'caret-right'
  | 'caret-down' | 'copy' | 'arrow-right' | 'arrow-left'
  | 'eye' | 'eye-off' | 'spinner' | 'dot';

interface Props extends Omit<SVGProps<SVGSVGElement>, 'name'> {
  name: Glyph;
  size?: number;
}

const PATHS: Record<Glyph, JSX.Element> = {
  bolt:        <path d="M13 2L4 14h6l-1 8 9-12h-6l1-8z" />,
  chip:        <g><rect x="6" y="6" width="12" height="12" /><path d="M9 2v3M12 2v3M15 2v3M9 19v3M12 19v3M15 19v3M2 9h3M2 12h3M2 15h3M19 9h3M19 12h3M19 15h3" /></g>,
  signal:      <path d="M3 12c4-6 14-6 18 0M6 14c3-4 9-4 12 0M9 16c2-2 4-2 6 0M11 18h2" />,
  cylinder:    <g><ellipse cx="12" cy="5" rx="8" ry="3" /><path d="M4 5v14c0 1.7 3.6 3 8 3s8-1.3 8-3V5" /><path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3" /></g>,
  branch:      <g><circle cx="6" cy="5" r="2" /><circle cx="6" cy="19" r="2" /><circle cx="18" cy="12" r="2" /><path d="M6 7v10M6 12c0-3 4-7 12-7" /></g>,
  braces:      <path d="M8 3c-2 0-3 1-3 3v3c0 1.5-1 2-2 2v2c1 0 2 0.5 2 2v3c0 2 1 3 3 3M16 3c2 0 3 1 3 3v3c0 1.5 1 2 2 2v2c-1 0-2 0.5-2 2v3c0 2-1 3-3 3" />,
  webhook:     <g><circle cx="6" cy="18" r="2.5" /><circle cx="18" cy="18" r="2.5" /><circle cx="12" cy="6" r="2.5" /><path d="M10 8l-4 7M14 8l4 7M8.5 18h7" /></g>,
  clock:       <g><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 3" /></g>,
  scroll:      <g><path d="M5 4h11l3 3v13H8c-1.7 0-3-1.3-3-3V4z" /><path d="M5 17h13M16 4v3h3" /></g>,
  key:         <g><circle cx="8" cy="12" r="4" /><path d="M12 12h10M18 12v4M22 12v4" /></g>,
  search:      <g><circle cx="11" cy="11" r="7" /><path d="M21 21l-5-5" /></g>,
  logout:      <g><path d="M9 4H5c-1.1 0-2 0.9-2 2v12c0 1.1 0.9 2 2 2h4" /><path d="M15 17l5-5-5-5M20 12H10" /></g>,
  plus:        <path d="M12 4v16M4 12h16" />,
  play:        <path d="M6 4l14 8-14 8V4z" />,
  stop:        <rect x="5" y="5" width="14" height="14" />,
  check:       <path d="M4 13l5 5L20 6" />,
  x:           <path d="M5 5l14 14M19 5L5 19" />,
  'caret-right': <path d="M9 5l7 7-7 7" />,
  'caret-down':  <path d="M5 9l7 7 7-7" />,
  copy:        <g><rect x="9" y="9" width="11" height="11" /><path d="M5 15V5c0-1.1 0.9-2 2-2h10" /></g>,
  'arrow-right': <path d="M5 12h14M13 6l6 6-6 6" />,
  'arrow-left':  <path d="M19 12H5M11 18l-6-6 6-6" />,
  eye:         <g><path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z" /><circle cx="12" cy="12" r="3" /></g>,
  'eye-off':   <g><path d="M3 3l18 18M10.6 10.6a3 3 0 004.2 4.2M9 5.5A10.9 10.9 0 0112 5c6 0 10 7 10 7a17 17 0 01-3.1 4.2M6.1 7.3A17 17 0 002 12s4 7 10 7c1.5 0 2.9-.3 4.2-.9" /></g>,
  spinner:     <g><circle cx="12" cy="12" r="9" opacity="0.25" /><path d="M21 12a9 9 0 00-9-9" /></g>,
  dot:         <circle cx="12" cy="12" r="4" />,
};

export default function Icon({ name, size = 14, strokeWidth = 1.5, ...rest }: Props) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={strokeWidth}
      strokeLinecap="square"
      strokeLinejoin="miter"
      aria-hidden="true"
      {...rest}
    >
      {PATHS[name]}
    </svg>
  );
}
