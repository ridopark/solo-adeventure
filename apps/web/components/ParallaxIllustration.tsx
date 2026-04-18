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

async function inspectDepth(url: string): Promise<void> {
  try {
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
    if (!ctx) return;
    ctx.drawImage(img, 0, 0);
    const data = ctx.getImageData(0, 0, canvas.width, canvas.height).data;
    let min = 255;
    let max = 0;
    let sum = 0;
    let n = 0;
    const stride = 64;
    for (let i = 0; i < data.length; i += 4 * stride) {
      const v = data[i];
      if (v < min) min = v;
      if (v > max) max = v;
      sum += v;
      n++;
    }
    const avg = Math.round(sum / n);
    console.log(
      `[parallax] depth stats ${canvas.width}x${canvas.height} min=${min} max=${max} avg=${avg} range=${max - min} (${max - min > 30 ? "OK" : "FLAT -- depth map has low variance"})`,
    );
  } catch (e) {
    console.warn("[parallax] depth inspect failed", e);
  }
}

const VERTEX_SHADER = /* glsl */ `
  uniform sampler2D depthMap;
  uniform float displacementScale;
  varying vec2 vUv;
  void main() {
    vUv = uv;
    float d = texture2D(depthMap, uv).r;
    vec3 displaced = position + vec3(0.0, 0.0, d * displacementScale);
    gl_Position = projectionMatrix * modelViewMatrix * vec4(displaced, 1.0);
  }
`;

const FRAGMENT_SHADER = /* glsl */ `
  uniform sampler2D colorMap;
  varying vec2 vUv;
  void main() {
    gl_FragColor = texture2D(colorMap, vUv);
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
    void inspectDepth(resolveURL(depthSrc));

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
        loadTexture(resolveURL(depthSrc), "depth"),
      ])
        .then(([colorTex, depthTex]) => {
          if (disposed) {
            colorTex.dispose();
            depthTex.dispose();
            return;
          }
          colorTex.colorSpace = THREE.SRGBColorSpace;
          depthTex.minFilter = THREE.LinearFilter;
          depthTex.magFilter = THREE.LinearFilter;

          const geometry = new THREE.PlaneGeometry(2, 2, 128, 128);
          const material = new THREE.ShaderMaterial({
            vertexShader: VERTEX_SHADER,
            fragmentShader: FRAGMENT_SHADER,
            uniforms: {
              colorMap: { value: colorTex },
              depthMap: { value: depthTex },
              displacementScale: { value: 0.9 },
            },
          });
          const mesh = new THREE.Mesh(geometry, material);
          scene.add(mesh);

          const phase = (seq % 4) * (Math.PI / 2);
          const start = performance.now();
          let frames = 0;
          let lastSample = start;
          console.log(`[parallax] render loop starting, phase=${phase.toFixed(2)} displacement=0.9`);
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
