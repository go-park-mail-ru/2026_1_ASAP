#!/usr/bin/env python3
"""CLIP zero-shot capybara detector. One-shot CLI or long-running --serve worker."""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path
from typing import Any

# Limit BLAS/thread pools so each worker stays within container RAM.
os.environ.setdefault("OMP_NUM_THREADS", "1")
os.environ.setdefault("MKL_NUM_THREADS", "1")
os.environ.setdefault("OPENBLAS_NUM_THREADS", "1")

import open_clip
import torch
from PIL import Image

LABELS = [
    "a photo of a capybara",
    "a photo of a rodent animal",
    "a photo of a dog or cat",
    "a photo of a person",
    "a photo of food",
]


def load_model(device: str = "cpu"):
    model, _, preprocess = open_clip.create_model_and_transforms(
        "ViT-B-32", pretrained="openai", device=device
    )
    tokenizer = open_clip.get_tokenizer("ViT-B-32")
    return model, preprocess, tokenizer


def classify_image(
    model,
    preprocess,
    tokenizer,
    image_path: Path,
    threshold: float,
) -> dict[str, Any]:
    device = "cpu"
    text = tokenizer(LABELS)
    with torch.no_grad():
        image = preprocess(Image.open(image_path).convert("RGB")).unsqueeze(0)
        image_features = model.encode_image(image)
        text_features = model.encode_text(text)
        image_features /= image_features.norm(dim=-1, keepdim=True)
        text_features /= text_features.norm(dim=-1, keepdim=True)
        probs = (100.0 * image_features @ text_features.T).softmax(dim=-1).squeeze(0)

    capy_score = float(probs[0].item())
    return {
        "is_capybara": capy_score >= threshold,
        "score": capy_score,
    }


def run_serve(default_threshold: float) -> int:
    model, preprocess, tokenizer = load_model()
    print(json.dumps({"ready": True}), flush=True)

    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
            image_path = Path(req["image_path"])
            threshold = float(req.get("threshold", default_threshold))
            payload = classify_image(model, preprocess, tokenizer, image_path, threshold)
        except Exception as exc:  # noqa: BLE001
            payload = {"is_capybara": False, "score": 0.0, "error": str(exc)}
        print(json.dumps(payload, ensure_ascii=False), flush=True)
    return 0


def run_once(image_path: Path, threshold: float) -> int:
    model, preprocess, tokenizer = load_model()
    payload = classify_image(model, preprocess, tokenizer, image_path, threshold)
    print(json.dumps(payload, ensure_ascii=False))
    return 0


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("image_path", type=Path, nargs="?")
    parser.add_argument("--threshold", type=float, default=0.28)
    parser.add_argument(
        "--serve",
        action="store_true",
        help="Load model once and read JSON requests from stdin (one response per line)",
    )
    args = parser.parse_args()

    if args.serve:
        return run_serve(args.threshold)

    if args.image_path is None:
        parser.error("image_path is required unless --serve is set")
    return run_once(args.image_path, args.threshold)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(json.dumps({"is_capybara": False, "score": 0.0, "error": str(exc)}))
        raise SystemExit(1)
