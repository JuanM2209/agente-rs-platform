"use client";

import { useEffect, useRef } from "react";

interface Point3D {
  x: number;
  y: number;
  z: number;
  pulsePhase: number;
}

const NUM_POINTS = 120;
const CONNECT_DIST = 0.52;
const ROTATION_SPEED = 0.0018;

function fibonacci_sphere(n: number): Point3D[] {
  const pts: Point3D[] = [];
  const golden = Math.PI * (3 - Math.sqrt(5));
  for (let i = 0; i < n; i++) {
    const y = 1 - (i / (n - 1)) * 2;
    const r = Math.sqrt(1 - y * y);
    const theta = golden * i;
    pts.push({ x: Math.cos(theta) * r, y, z: Math.sin(theta) * r, pulsePhase: Math.random() * Math.PI * 2 });
  }
  return pts;
}

export function NetworkSphere() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const pts = fibonacci_sphere(NUM_POINTS);
    let angle = 0;
    let raf: number;

    function resize() {
      if (!canvas) return;
      canvas.width = canvas.offsetWidth * devicePixelRatio;
      canvas.height = canvas.offsetHeight * devicePixelRatio;
    }
    resize();
    const ro = new ResizeObserver(resize);
    ro.observe(canvas);

    function project(p: Point3D, cosA: number, sinA: number, w: number, h: number) {
      // Rotate around Y axis
      const rx = p.x * cosA + p.z * sinA;
      const ry = p.y;
      const rz = -p.x * sinA + p.z * cosA;
      // Perspective
      const fov = 2.8;
      const scale = fov / (fov + rz + 1);
      const sx = w / 2 + rx * scale * (w * 0.38);
      const sy = h / 2 + ry * scale * (h * 0.38);
      return { sx, sy, scale, rz };
    }

    function draw(t: number) {
      if (!canvas || !ctx) return;
      const w = canvas.width;
      const h = canvas.height;
      ctx.clearRect(0, 0, w, h);

      angle += ROTATION_SPEED;
      const cosA = Math.cos(angle);
      const sinA = Math.sin(angle);

      // Project all points
      const projected = pts.map((p) => ({ ...project(p, cosA, sinA, w, h), pulse: p.pulsePhase }));

      // Draw connections
      for (let i = 0; i < pts.length; i++) {
        for (let j = i + 1; j < pts.length; j++) {
          const dx = pts[i].x - pts[j].x;
          const dy = pts[i].y - pts[j].y;
          const dz = pts[i].z - pts[j].z;
          const dist = Math.sqrt(dx * dx + dy * dy + dz * dz);
          if (dist > CONNECT_DIST) continue;

          const pi = projected[i];
          const pj = projected[j];
          const avgDepth = (pi.rz + pj.rz) / 2;
          const depthAlpha = (avgDepth + 1) / 2; // 0..1
          const lineAlpha = (1 - dist / CONNECT_DIST) * depthAlpha * 0.35;

          ctx.beginPath();
          ctx.moveTo(pi.sx, pi.sy);
          ctx.lineTo(pj.sx, pj.sy);
          ctx.strokeStyle = `rgba(173, 198, 255, ${lineAlpha})`;
          ctx.lineWidth = 0.6;
          ctx.stroke();
        }
      }

      // Draw nodes
      projected.forEach((p, i) => {
        const depthAlpha = (p.rz + 1) / 2;
        const pulse = 0.5 + 0.5 * Math.sin(t * 0.001 + pts[i].pulsePhase);
        const r = (1.5 + pulse * 1.5) * p.scale * devicePixelRatio;
        const alpha = 0.3 + depthAlpha * 0.7;

        // Glow
        const grad = ctx.createRadialGradient(p.sx, p.sy, 0, p.sx, p.sy, r * 3);
        const isSpecial = i % 15 === 0;
        const color = isSpecial ? "78, 222, 163" : "173, 198, 255";
        grad.addColorStop(0, `rgba(${color}, ${alpha})`);
        grad.addColorStop(1, `rgba(${color}, 0)`);
        ctx.beginPath();
        ctx.arc(p.sx, p.sy, r * 3, 0, Math.PI * 2);
        ctx.fillStyle = grad;
        ctx.fill();

        // Core dot
        ctx.beginPath();
        ctx.arc(p.sx, p.sy, r, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(${color}, ${alpha})`;
        ctx.fill();
      });

      raf = requestAnimationFrame(draw);
    }

    raf = requestAnimationFrame(draw);
    return () => {
      cancelAnimationFrame(raf);
      ro.disconnect();
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      className="w-full h-full"
      style={{ display: "block" }}
    />
  );
}
