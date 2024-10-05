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

app.use(expressConnectMiddleware({
  routes
}));

app.get("/", function (req, res) {
  res.send("hello, world!");
});

http.createServer(app).listen(8080);