"use client";

import { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import { BACKEND_URL } from "@/lib/env";

function resolveURL(url: string) {
  return url.startsWith("http") ? url : `${BACKEND_URL}${url}`;
}

// Cross-origin image URLs can't be used as WebGL textures without CORS, so
// route them through the backend proxy which returns the same bytes with
// same-origin permissions.
function proxyImage(url: string) {
  if (!url.startsWith("http")) return `${BACKEND_URL}${url}`;
  return `${BACKEND_URL}/img?url=${encodeURIComponent(url)}`;
}

interface DepthAnalysis {
  texture: THREE.Texture;
  width: number;
  height: number;
  range: number; // 0-255 depth span
  avgGradient: number; // 0-1 average edge magnitude
  edgeRatio: number; // 0-1 fraction of pixels at sharp cliffs
}

// Auto-tune displacement scale per image. Dense sharp edges tear vertices,
// so we pull back hard when the depth map is busy; smooth maps get more pop.
function autoDisplacement(a: DepthAnalysis): number {
  if (a.range < 30) return 0.15;
  const base = 0.75 - a.edgeRatio * 3.2;
  return Math.max(0.2, Math.min(0.75, base));
}

// Load the depth image once, analyze pixels, and reuse the element for the
// three.js texture so we don't fetch it twice.
async function analyzeDepth(url: string): Promise<DepthAnalysis> {
  const img = await new Promise<HTMLImageElement>((resolve, reject) => {
    const el = new Image();
    el.crossOrigin = "anonymous";
    el.onload = () => resolve(el);
    el.onerror = reject;
    el.src = url;
  });
  const canvas = document.createElement("canvas");
  canvas.width = img.naturalWidth;
  canvas.height = img.naturalHeight;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("no 2d context");
  ctx.drawImage(img, 0, 0);
  const { width: W, height: H } = canvas;
  const data = ctx.getImageData(0, 0, W, H).data;

  let min = 255;
  let max = 0;
  let totalGradient = 0;
  let sharpEdges = 0;
  let samples = 0;
  const step = 4;
  const rowStride = W * 4;
  for (let y = step; y < H - step; y += step) {
    for (let x = step; x < W - step; x += step) {
      const i = y * rowStride + x * 4;
      const v = data[i];
      if (v < min) min = v;
      if (v > max) max = v;
      const l = data[i - step * 4];
      const r = data[i + step * 4];
      const u = data[i - step * rowStride];
      const d = data[i + step * rowStride];
      const g = Math.max(Math.abs(r - l), Math.abs(d - u));
      totalGradient += g;
      if (g > 48) sharpEdges++;
      samples++;
    }
  }
  const avgGradient = totalGradient / samples / 255;
  const edgeRatio = sharpEdges / samples;

  const texture = new THREE.Texture(img);
  texture.needsUpdate = true;
  return {
    texture,
    width: W,
    height: H,
    range: max - min,
    avgGradient,
    edgeRatio,
  };
}

const VERTEX_SHADER = /* glsl */ `
  uniform sampler2D depthMap;
  uniform float displacementScale;
  uniform vec2 texelSize;
  varying vec2 vUv;
  varying float vDepthGradient;

  // 3x3 box blur of the depth sample smooths sharp depth cliffs so
  // vertices don't tear across foreground/background transitions.
  float sampleDepth(vec2 uv) {
    float sum = 0.0;
    for (int i = -1; i <= 1; i++) {
      for (int j = -1; j <= 1; j++) {
        sum += texture2D(depthMap, uv + vec2(float(i), float(j)) * texelSize).r;
      }
    }
    return sum / 9.0;
  }

  void main() {
    vUv = uv;
    float d = sampleDepth(uv);
    // soften extreme displacement with a gentle curve
    float softened = pow(d, 0.85);
    // measure local gradient so the fragment shader can fade stretched geometry
    float dx = abs(sampleDepth(uv + vec2(texelSize.x, 0.0)) - sampleDepth(uv - vec2(texelSize.x, 0.0)));
    float dy = abs(sampleDepth(uv + vec2(0.0, texelSize.y)) - sampleDepth(uv - vec2(0.0, texelSize.y)));
    vDepthGradient = max(dx, dy);

    vec3 displaced = position + vec3(0.0, 0.0, softened * displacementScale);
    gl_Position = projectionMatrix * modelViewMatrix * vec4(displaced, 1.0);
  }
`;

const FRAGMENT_SHADER = /* glsl */ `
  uniform sampler2D colorMap;
  varying vec2 vUv;
  varying float vDepthGradient;
  void main() {
    vec4 c = texture2D(colorMap, vUv);
    // fade stretched triangles where depth changes fast (edges between
    // foreground and background) to hide the rubber-sheeting artifact
    float edgeFade = 1.0 - smoothstep(0.04, 0.12, vDepthGradient);
    gl_FragColor = vec4(c.rgb, c.a * edgeFade);
  }
`;

export function ParallaxIllustration({
  imageSrc,
  depthSrc,
  alt,
  seq = 0,
}: {
  imageSrc: string;
  depthSrc: string;
  alt: string;
  seq?: number;
}) {
  const mountRef = useRef<HTMLDivElement | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const mount = mountRef.current;
    if (!mount) return;

    console.log(`[parallax] init seq=${seq} image=${imageSrc} depth=${depthSrc}`);

    let renderer: THREE.WebGLRenderer | null = null;
    let animationId = 0;
    let resizeObserver: ResizeObserver | null = null;
    let disposed = false;

    try {
      const width = mount.clientWidth || 512;
      const height = mount.clientHeight || 512;

      const scene = new THREE.Scene();
      const camera = new THREE.PerspectiveCamera(50, width / height, 0.1, 10);
      camera.position.z = 2.2;

      renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
      renderer.setSize(width, height, false);
      mount.appendChild(renderer.domElement);
      console.log(`[parallax] webgl context created ${width}x${height} dpr=${renderer.getPixelRatio()}`);

      const loader = new THREE.TextureLoader();
      loader.setCrossOrigin("anonymous");

      const loadTexture = (url: string, label: string) =>
        new Promise<THREE.Texture>((resolve, reject) => {
          const t0 = performance.now();
          loader.load(
            url,
            (t) => {
              const ms = Math.round(performance.now() - t0);
              const img = t.image as { width?: number; height?: number } | undefined;
              console.log(`[parallax] texture loaded: ${label} ${img?.width}x${img?.height} in ${ms}ms`);
              resolve(t);
            },
            undefined,
            (e) => {
              console.error(`[parallax] texture load failed: ${label}`, e);
              reject(e);
            },
          );
        });

      Promise.all([
        loadTexture(proxyImage(imageSrc), "color"),
        analyzeDepth(resolveURL(depthSrc)),
      ])
        .then(([colorTex, depthAnalysis]) => {
          const depthTex = depthAnalysis.texture;
          if (disposed) {
            colorTex.dispose();
            depthTex.dispose();
            return;
          }
          colorTex.colorSpace = THREE.SRGBColorSpace;
          depthTex.minFilter = THREE.LinearFilter;
          depthTex.magFilter = THREE.LinearFilter;
          const dispScale = autoDisplacement(depthAnalysis);
          console.log(
            `[parallax] depth stats ${depthAnalysis.width}x${depthAnalysis.height} range=${depthAnalysis.range} avgGrad=${depthAnalysis.avgGradient.toFixed(3)} edgeRatio=${depthAnalysis.edgeRatio.toFixed(3)} -> displacement=${dispScale.toFixed(2)}`,
          );

          const geometry = new THREE.PlaneGeometry(2, 2, 200, 200);
          const tw = depthAnalysis.width;
          const th = depthAnalysis.height;
          const material = new THREE.ShaderMaterial({
            vertexShader: VERTEX_SHADER,
            fragmentShader: FRAGMENT_SHADER,
            uniforms: {
              colorMap: { value: colorTex },
              depthMap: { value: depthTex },
              displacementScale: { value: dispScale },
              texelSize: { value: new THREE.Vector2(1 / tw, 1 / th) },
            },
            transparent: true,
          });
          const mesh = new THREE.Mesh(geometry, material);
          scene.add(mesh);

          const phase = (seq % 4) * (Math.PI / 2);
          const start = performance.now();
          let frames = 0;
          let lastSample = start;
          console.log(`[parallax] render loop starting, phase=${phase.toFixed(2)} displacement=${dispScale.toFixed(2)} subdiv=200 edgeFade=on texel=${(1 / tw).toFixed(5)}`);
          const render = () => {
            if (disposed) return;
            const now = performance.now();
            const t = (now - start) * 0.0006;
            camera.position.x = Math.sin(t + phase) * 0.45;
            camera.position.y = Math.cos(t * 1.2 + phase) * 0.25;
            camera.lookAt(0, 0, 0);
            renderer!.render(scene, camera);
            frames++;
            if (now - lastSample > 3000) {
              console.log(
                `[parallax] rendering ${(frames / ((now - lastSample) / 1000)).toFixed(1)} fps, cam=(${camera.position.x.toFixed(2)}, ${camera.position.y.toFixed(2)}, ${camera.position.z.toFixed(2)})`,
              );
              frames = 0;
              lastSample = now;
            }
            animationId = requestAnimationFrame(render);
          };
          render();

          const onResize = () => {
            if (!mount || !renderer) return;
            const w = mount.clientWidth;
            const h = mount.clientHeight;
            if (w === 0 || h === 0) return;
            renderer.setSize(w, h, false);
            camera.aspect = w / h;
            camera.updateProjectionMatrix();
          };
          resizeObserver = new ResizeObserver(onResize);
          resizeObserver.observe(mount);

          (mount as unknown as { __cleanup: () => void }).__cleanup = () => {
            geometry.dispose();
            material.dispose();
            colorTex.dispose();
            depthTex.dispose();
          };
        })
        .catch((e) => {
          console.error("[parallax] falling back to still image", e);
          setFailed(true);
        });
    } catch (e) {
      console.error("[parallax] webgl setup threw", e);
      setFailed(true);
    }

    return () => {
      disposed = true;
      console.log(`[parallax] cleanup seq=${seq}`);
      if (animationId) cancelAnimationFrame(animationId);
      resizeObserver?.disconnect();
      const cleanup = (mount as unknown as { __cleanup?: () => void }).__cleanup;
      if (cleanup) cleanup();
      if (renderer) {
        renderer.dispose();
        if (renderer.domElement.parentNode) {
          renderer.domElement.parentNode.removeChild(renderer.domElement);
        }
      }
    };
  }, [imageSrc, depthSrc, seq]);

  if (failed) {
    return (
      <div className="aspect-square rounded-md border border-stone-300 shadow-inner bg-stone-200 overflow-hidden">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src={resolveURL(imageSrc)} alt={alt} className="w-full h-full object-cover" />
      </div>
    );
  }

  return (
    <div
      ref={mountRef}
      aria-label={alt}
      className="aspect-square rounded-md border border-stone-300 shadow-inner bg-stone-200 overflow-hidden"
    />
  );
}
