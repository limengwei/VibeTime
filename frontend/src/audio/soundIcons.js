const icons = {
  rain: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2.69l5.66 5.66a8 8 0 1 1-11.31 0z"/></svg>`,
  thunder: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M19 16.9A5 5 0 0 0 18 7h-1.26a8 8 0 1 0-11.62 9"/><polyline points="13 11 9 17 15 17 11 23"/></svg>`,
  wind: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9.59 4.59A2 2 0 1 1 11 8H2m10.59 11.41A2 2 0 1 0 14 16H2m15.73-8.27A2.5 2.5 0 1 1 19.5 12H2"/></svg>`,
  fire: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 12c2-2.96 0-7-1-8 0 3.038-1.773 4.741-3 6-1.226 1.26-2 3.24-2 5a6 6 0 1 0 12 0c0-1.532-1.056-3.94-2-5-1.786 3-2.791 3-4 2z"/></svg>`,
  ocean: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 6c.6.5 1.2 1 2.5 1C7 7 7 5 9.5 5c2.6 0 2.4 2 5 2 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1"/><path d="M2 12c.6.5 1.2 1 2.5 1 2.5 0 2.5-2 5-2 2.6 0 2.4 2 5 2 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1"/><path d="M2 18c.6.5 1.2 1 2.5 1 2.5 0 2.5-2 5-2 2.6 0 2.4 2 5 2 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1"/></svg>`,
  birds: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M16 7a4 4 0 1 1-8 0"/><path d="M12 11v4"/><path d="M8 18a4 4 0 0 1 8 0"/><line x1="12" y1="2" x2="12" y2="4"/></svg>`,
  whiteNoise: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12h2l3-9 4 18 4-18 3 9h2"/></svg>`,
  cafe: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M18 8h1a4 4 0 0 1 0 8h-1"/><path d="M2 8h16v9a4 4 0 0 1-4 4H6a4 4 0 0 1-4-4V8z"/><line x1="6" y1="1" x2="6" y2="4"/><line x1="10" y1="1" x2="10" y2="4"/><line x1="14" y1="1" x2="14" y2="4"/></svg>`,
  forest: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M17 14l-5-10-5 10"/><path d="M7 14l-3 6h16l-3-6"/><path d="M12 4v2"/></svg>`,
  singingBowl: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/><circle cx="12" cy="12" r="2"/></svg>`,
  fan: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M12 1v6m0 6v6m-7.5-3l5.2-3m4.6-2.5L19.5 7M1 12h6m6 0h6m-3 7.5l-3-5.2M9.5 11.1L4.5 4.5"/></svg>`,
  river: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12c2-2 4-2 6 0s4 2 6 0 4-2 6 0"/><path d="M2 18c2-2 4-2 6 0s4 2 6 0 4-2 6 0"/><path d="M2 6c2-2 4-2 6 0s4 2 6 0 4-2 6 0"/></svg>`,
  waterfall: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M8 2v8l-3 12h14L16 10V2"/><line x1="8" y1="2" x2="16" y2="2"/><path d="M10 6h4"/></svg>`,
  jungle: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L6 10h3l-3 6h3l-3 6h12l-3-6h3l-3-6h3L12 2z"/></svg>`,
  snow: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v20M2 12h20M4.93 4.93l14.14 14.14M19.07 4.93L4.93 19.07"/></svg>`,
  leaves: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M11 20A7 7 0 0 1 9.8 6.9C15.5 4.9 17 3.5 19 2c1 2 2 4.5 2 8 0 5.5-4.78 10-10 10z"/><path d="M2 21c0-3 1.85-5.36 5.08-6C9.5 14.52 12 13 13 12"/></svg>`,
  droplets: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2.69l5.66 5.66a8 8 0 1 1-11.31 0z"/></svg>`,
  cat: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22c5.523 0 10-4.477 10-10 0-5.523-4.477-10-10-10S2 6.477 2 12c0 5.523 4.477 10 10 10z"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/></svg>`,
  dog: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10 5.172C10 3.782 8.423 2.679 6.5 3c-2.823.47-4.113 6.006-4 7 .08.703 1.725 1.722 3.656 1 1.261-.472 1.96-1.45 2.344-2.5"/><path d="M14.267 5.172c0-1.39 1.577-2.493 3.5-2.172 2.823.47 4.113 6.006 4 7-.08.703-1.725 1.722-3.656 1-1.261-.472-1.855-1.45-2.239-2.5"/><path d="M8 14v.5"/><path d="M16 14v.5"/><path d="M11.25 16.25h1.5L12 17l-.75-.75z"/><path d="M4.42 11.247A13.152 13.152 0 0 0 4 14.556C4 18.728 7.582 21 12 21s8-2.272 8-6.444c0-1.061-.162-2.2-.493-3.309m-9.243-6.082A8.801 8.801 0 0 1 12 5c.78 0 1.5.108 2.161.306"/></svg>`,
  frog: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="8" r="2"/><circle cx="16" cy="8" r="2"/><path d="M4 12c0 4.418 3.582 8 8 8s8-3.582 8-8"/><path d="M6 6L2 2"/><path d="M18 6l4-4"/></svg>`,
  horse: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12c0-3-2.5-6-7-6-1.5 0-3 .5-4 1.5L9 10l-3 1-2 4 3 2v5h4v-3l3 1 2 2h3l-1-4 3-2z"/><circle cx="15" cy="9" r="1" fill="currentColor"/></svg>`,
  whale: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12c2-4 6-7 10-7s8 3 10 7c-2 4-6 7-10 7S4 16 2 12z"/><path d="M7 10h.01M17 10h.01"/><path d="M12 14c-2 0-4-1-5-2"/></svg>`,
  wolf: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L8 6H4l2 4-2 6h4l4 6 4-6h4l-2-6 2-4h-4L12 2z"/></svg>`,
  owl: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="10" r="8"/><circle cx="9" cy="10" r="2"/><circle cx="15" cy="10" r="2"/><path d="M12 18v4M8 22h8"/></svg>`,
  cricket: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M8 4c0 0-3 4-3 8s3 8 3 8"/><path d="M16 4c0 0 3 4 3 8s-3 8-3 8"/><circle cx="12" cy="12" r="2"/></svg>`,
  seagull: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 10c3-3 6-3 10 0s7 3 10 0"/><path d="M12 10v6"/><path d="M8 20l4-4 4 4"/></svg>`,
  beehive: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2l4 6h4l-2 6 2 6H6l2-6-2-6h4L12 2z"/></svg>`,
  woodpecker: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v10m0 0l-4 4m4-4l4 4"/><circle cx="12" cy="20" r="2"/></svg>`,
  sheep: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="8" r="6"/><path d="M6 14c-2 0-3 2-3 4h18c0-2-1-4-3-4"/><circle cx="10" cy="8" r="1" fill="currentColor"/><circle cx="14" cy="8" r="1" fill="currentColor"/></svg>`,
  cow: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="6" width="16" height="10" rx="4"/><path d="M4 10H2m20 0h-2"/><circle cx="9" cy="11" r="1" fill="currentColor"/><circle cx="15" cy="11" r="1" fill="currentColor"/><path d="M10 15h4"/></svg>`,
  chicken: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="10" r="6"/><path d="M12 4V2"/><path d="M9 16l-1 6m7-6l1 6"/><circle cx="10" cy="9" r="1" fill="currentColor"/></svg>`,
  crow: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 8l6 4-2 8h12l-2-8 6-4"/><circle cx="12" cy="6" r="3"/></svg>`,
  binaural: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M6 12a6 6 0 0 1 0-4m12 4a6 6 0 0 0 0-4"/><path d="M3 12a9 9 0 0 1 0-6m18 6a9 9 0 0 0 0-6"/></svg>`,
  airplane: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M17.8 19.2L16 11l3.5-3.5C21 6 21.5 4 21 3c-1-.5-3 0-4.5 1.5L13 8 4.8 6.2c-.5-.1-.9.1-1.1.5l-.3.5c-.2.5-.1 1 .3 1.3L9 12l-2 3H4l-1 1 3 2 2 3 1-1v-3l3-2 3.5 5.3c.3.4.8.5 1.3.3l.5-.2c.4-.3.6-.7.5-1.2z"/></svg>`,
  train: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="4" y="3" width="16" height="16" rx="2"/><path d="M4 11h16"/><path d="M12 3v8"/><path d="M8 19l-2 3m10-3l2 3"/><circle cx="8" cy="15" r="1" fill="currentColor"/><circle cx="16" cy="15" r="1" fill="currentColor"/></svg>`,
  boat: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 18l4-8h12l4 8"/><path d="M12 2v8"/><path d="M8 10l4-8 4 8"/></svg>`,
  submarine: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="10" width="20" height="6" rx="3"/><path d="M12 10V6"/><path d="M8 6h8"/><circle cx="7" cy="13" r="1" fill="currentColor"/><circle cx="17" cy="13" r="1" fill="currentColor"/></svg>`,
  siren: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v4m0 0c-4 0-6 3-6 6v4h12v-4c0-3-2-6-6-6z"/><rect x="4" y="16" width="16" height="4" rx="1"/></svg>`,
  street: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 20V4m16 16V4M8 12h8"/><path d="M4 12l2-4m14 4l-2-4"/></svg>`,
  crowd: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="7" cy="6" r="3"/><circle cx="17" cy="6" r="3"/><path d="M3 20c0-3 3-5 4-5h0"/><path d="M21 20c0-3-3-5-4-5h0"/><circle cx="12" cy="8" r="3"/><path d="M8 21c0-3 2-5 4-5s4 2 4 5"/></svg>`,
  fireworks: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v4m0 0l-3 3m3-3l3 3m-3-3c-4 0-6 3-6 6m6-6c4 0 6 3 6 6"/><circle cx="12" cy="15" r="1" fill="currentColor"/><circle cx="8" cy="17" r="1" fill="currentColor"/><circle cx="16" cy="17" r="1" fill="currentColor"/><circle cx="12" cy="19" r="1" fill="currentColor"/></svg>`,
  highway: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 20L8 4m8 0l4 16"/><path d="M8 12h8"/></svg>`,
  traffic: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="8" y="2" width="8" height="20" rx="2"/><circle cx="12" cy="7" r="1.5" fill="currentColor"/><circle cx="12" cy="12" r="1.5" fill="currentColor"/><circle cx="12" cy="17" r="1.5" fill="currentColor"/></svg>`,
  airport: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 20h20M4 16l6-4v-4l4-2v6l6-2v4l-6 2H4z"/></svg>`,
  library: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/><line x1="8" y1="7" x2="16" y2="7"/><line x1="8" y1="11" x2="14" y2="11"/></svg>`,
  office: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M2 7l10-5 10 5"/><line x1="8" y1="11" x2="8" y2="11.01"/><line x1="12" y1="11" x2="12" y2="11.01"/><line x1="16" y1="11" x2="16" y2="11.01"/></svg>`,
  restaurant: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 2v7c0 1.1.9 2 2 2h4a2 2 0 0 0 2-2V2"/><path d="M7 2v20"/><path d="M21 15V2v0a5 5 0 0 0-5 5v6c0 1.1.9 2 2 2h3zm0 0v7"/></svg>`,
  underwater: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12c2-2 4-2 6 0s4 2 6 0 4-2 6 0"/><path d="M2 7c2-2 4-2 6 0s4 2 6 0 4-2 6 0"/><circle cx="16" cy="17" r="3"/></svg>`,
  church: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L6 8v14h12V8L12 2z"/><path d="M12 2v6m0 0H6m6 0h6"/><path d="M10 22v-6h4v6"/></svg>`,
  temple: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h16"/><path d="M6 18h12"/><path d="M8 14h8"/><path d="M10 10h4"/><path d="M12 2v8"/><path d="M6 14l6-12 6 12"/></svg>`,
  carousel: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M8 12a4 4 0 0 1 8 0"/><path d="M2 12h2m16 0h2"/></svg>`,
  construction: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 20h20M5 20V8l7-5 7 5v12"/><line x1="9" y1="12" x2="9" y2="12.01"/><line x1="15" y1="12" x2="15" y2="12.01"/></svg>`,
  bar: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="10" rx="1"/><path d="M12 3v8"/><path d="M8 7l4-4 4 4"/></svg>`,
  lab: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9 3h6v5l5 10H4L9 8V3z"/><line x1="6" y1="14" x2="18" y2="14"/></svg>`,
  laundry: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="13" r="5"/><rect x="3" y="2" width="18" height="20" rx="2"/><circle cx="8" cy="6" r="1" fill="currentColor"/><circle cx="12" cy="6" r="1" fill="currentColor"/></svg>`,
  village: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 21h18"/><path d="M5 21V7l8-4v18"/><path d="M19 21V11l-6-4"/><path d="M9 9v.01M9 12v.01M9 15v.01M9 18v.01"/></svg>`,
  subway: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="6" y="3" width="12" height="16" rx="2"/><path d="M6 11h12"/><circle cx="9" cy="15" r="1" fill="currentColor"/><circle cx="15" cy="15" r="1" fill="currentColor"/><path d="M8 19l-2 3m10-3l2 3"/></svg>`,
  supermarket: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M6 2L3 7v13a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V7l-3-5z"/><line x1="3" y1="7" x2="21" y2="7"/><path d="M16 11a4 4 0 0 1-8 0"/></svg>`,
  boiling: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M6 10h12v8a6 6 0 0 1-12 0v-8z"/><path d="M8 6c0-1 .5-2 1.5-2S11 5 12 5s1.5-1 2.5-1S16 5 16 6"/><path d="M9 14v.01M12 13v.01M15 14v.01"/></svg>`,
  bubbles: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="16" r="4"/><circle cx="16" cy="12" r="3"/><circle cx="12" cy="6" r="2"/></svg>`,
  clock: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>`,
  dryer: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="6"/><rect x="3" y="2" width="18" height="20" rx="2"/><circle cx="12" cy="12" r="2"/><circle cx="7" cy="5" r="1" fill="currentColor"/></svg>`,
  keyboard: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="6" width="20" height="12" rx="2"/><line x1="6" y1="10" x2="6" y2="10.01"/><line x1="10" y1="10" x2="10" y2="10.01"/><line x1="14" y1="10" x2="14" y2="10.01"/><line x1="18" y1="10" x2="18" y2="10.01"/><line x1="8" y1="14" x2="16" y2="14"/></svg>`,
  morse: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="4" cy="12" r="2" fill="currentColor"/><circle cx="12" cy="12" r="2" fill="currentColor"/><line x1="18" y1="8" x2="18" y2="16"/></svg>`,
  paper: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>`,
  projector: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="8" width="20" height="10" rx="2"/><circle cx="12" cy="13" r="3"/><path d="M7 2l5 6m5-6l-5 6"/></svg>`,
  radio: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 9V2m0 7h16a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2v-8a1 1 0 0 1 1-1z"/><circle cx="15" cy="15" r="3"/><line x1="6" y1="13" x2="6" y2="13.01"/><line x1="6" y1="17" x2="6" y2="17.01"/></svg>`,
  typewriter: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="10" width="20" height="10" rx="2"/><path d="M6 10V6a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v4"/><line x1="6" y1="14" x2="6.01" y2="14"/><line x1="10" y1="14" x2="10.01" y2="14"/><line x1="14" y1="14" x2="14.01" y2="14"/><line x1="18" y1="14" x2="18.01" y2="14"/><path d="M8 18h8"/></svg>`,
  vinyl: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3"/><circle cx="12" cy="12" r="1" fill="currentColor"/></svg>`,
  washing: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="13" r="5"/><rect x="3" y="2" width="18" height="20" rx="2"/><path d="M3 8h18"/><circle cx="7" cy="5" r="1" fill="currentColor"/><circle cx="11" cy="5" r="1" fill="currentColor"/></svg>`,
  windChimes: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v4"/><path d="M8 6h8"/><path d="M8 6l-1 8"/><path d="M16 6l1 8"/><path d="M10 14l-1 6"/><path d="M14 14l1 6"/><circle cx="12" cy="14" r="2"/></svg>`,
  pinkNoise: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12h3l2-4 3 8 3-8 3 8 2-4h4"/></svg>`,
  road: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M4 20L8 4m8 0l4 16"/><path d="M12 6v2m0 4v2m0 4v2"/></svg>`,
  night: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>`
}

const categoryMap = {
  animals: 'dog',
  binaural: 'binaural',
  nature: 'forest',
  noise: 'whiteNoise',
  places: 'office',
  rain: 'rain',
  things: 'clock',
  transport: 'train',
  urban: 'street'
}

export function getIconSvg(name) {
  if (icons[name]) return icons[name]

  const baseName = name.includes('/') ? name.split('/').pop() : name

  if (icons[baseName]) return icons[baseName]

  for (const key of Object.keys(icons)) {
    if (baseName.includes(key) || baseName.startsWith(key)) {
      return icons[key]
    }
  }

  for (const [prefix, iconKey] of Object.entries(categoryMap)) {
    if (name.startsWith(prefix + '/')) {
      return icons[iconKey]
    }
  }

  return icons.whiteNoise
}
