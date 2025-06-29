import http from "http";
import express from "express";
import "dotenv/config";
import routes from "./connect";
import * as falProxy from "@fal-ai/serverless-proxy/express";
import cors from "cors";
import morgan from "morgan";

import { expressConnectMiddleware } from "@connectrpc/connect-express";

const app = express();
app.use(express.json());
app.use(morgan("combined"));
app.all(falProxy.route, cors(), falProxy.handler);

app.use(
  expressConnectMiddleware({
    routes,
  }),
);

app.get("/healthz", function (_req, res) {
  res.send("OK");
});

console.log("Listening on port 8080");

http.createServer(app).listen(8080);
