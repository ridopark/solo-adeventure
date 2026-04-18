"use client";

import { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import { BACKEND_URL } from "@/lib/env";

function resolveURL(url: string) {
  return url.startsWith("http") ? url : `${BACKEND_URL}${url}`;
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

    let renderer: THREE.WebGLRenderer | null = null;
    let animationId = 0;
    let resizeObserver: ResizeObserver | null = null;
    let disposed = false;

    try {
      const width = mount.clientWidth || 512;
      const height = mount.clientHeight || 512;

      const scene = new THREE.Scene();
      const camera = new THREE.PerspectiveCamera(40, width / height, 0.1, 10);
      camera.position.z = 2.6;

      renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
      renderer.setSize(width, height, false);
      mount.appendChild(renderer.domElement);

      const loader = new THREE.TextureLoader();
      loader.setCrossOrigin("anonymous");

      const loadTexture = (url: string) =>
        new Promise<THREE.Texture>((resolve, reject) => {
          loader.load(url, (t) => resolve(t), undefined, (e) => reject(e));
        });

      Promise.all([loadTexture(resolveURL(imageSrc)), loadTexture(resolveURL(depthSrc))])
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
              displacementScale: { value: 0.35 },
            },
          });
          const mesh = new THREE.Mesh(geometry, material);
          scene.add(mesh);

          const phase = (seq % 4) * (Math.PI / 2);
          const start = performance.now();
          const render = () => {
            if (disposed) return;
            const t = (performance.now() - start) * 0.0006;
            camera.position.x = Math.sin(t + phase) * 0.22;
            camera.position.y = Math.cos(t * 1.2 + phase) * 0.12;
            camera.lookAt(0, 0, 0);
            renderer!.render(scene, camera);
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
        .catch(() => {
          setFailed(true);
        });
    } catch {
      setFailed(true);
    }

    return () => {
      disposed = true;
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
