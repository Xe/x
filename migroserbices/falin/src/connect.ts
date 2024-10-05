import { ConnectRouter } from "@connectrpc/connect";
import * as fal from "@fal-ai/serverless-client";
import { ImageService } from "../gen/falin_connect";
import { GenerateImageResponse, ImageOutput } from "../gen/falin_pb";

interface falImageOutput {
  url: string;
  width: number;
  height: number;
  content_type: string;
}

interface falOutput {
  images: falImageOutput[];
  seed: number;
  prompt: string;
}

export default function (router: ConnectRouter) {
  router.rpc(ImageService, ImageService.methods.generateImage, async (req): Promise<GenerateImageResponse> => {
    try {
      const result: falOutput = await fal.run(req.model, {
        input: {
          prompt: req.prompt,
          seed: req.seed || undefined,
          image_size: {
            width: 1344,
            height: 768,
          },
          num_images: req.numImages || 1,
          enable_safety_checker: req.enableSafetyChecker || true,
        }
      });

      const images = result.images.map(img => new ImageOutput({
        url: img.url,
        width: img.width,
        height: img.height,
        contentType: img.content_type,
      }));

      return new GenerateImageResponse({
        images,
        prompt: result.prompt,
      });
    }
    catch (e) {
      console.log(e);
      throw e;
    }
  });
}