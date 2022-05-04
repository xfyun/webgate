
package com.iflytek.webapi.iat;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import okhttp3.*;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.io.InputStream;
import java.net.URL;
import java.net.URLEncoder;
import java.nio.charset.Charset;
import java.text.SimpleDateFormat;
import java.util.*;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * <p>Title : 流式服务通用demo</p>
 * <p>Description : websocket通用demo</p>
 * <p>Date : 2020/8/4 </p>
 *
 * @author : hejie
 */
public class WebsocketDemo extends WebSocketListener {
    private static final String REQUEST_URL = "ws://ws-api.xf-yun.com/v1/private/s0bd29636";
    private static final String API_KEY = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx";
    private static final String API_SECRET = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx";
    private static final String appId = "xxxxxxxxxx";
    /**
     * status 枚举
     */
    public static final int STATUS_BEGIN = 0;
    public static final int STATUS_CONTINUE = 1;
    public static final int STATUS_END = 2;
    public static final int STATUS_STOP = 4;

    private LinkedBlockingQueue<String> inputDataQueue = new LinkedBlockingQueue<>();  // 存放请求输入数据队列
    private AtomicBoolean  isClose = new AtomicBoolean(false);



    private InputStream inputStream;
    public WebSocket webSocket;

    public WebsocketDemo(InputStream inputStream) {
        this.inputStream = inputStream;
    }

    public static void main(String[] args) throws FileNotFoundException {
        WebsocketDemo demo = new WebsocketDemo(new FileInputStream("/Users/sjliu/Downloads/ppp.opus-wb"));
        demo.doRequest();
    }

    private void doRequest() {
        try {
            OkHttpClient client = new OkHttpClient.Builder().build();
            //将url中的 schema http://和https://分别替换为ws:// 和 wss://
            String authedRequestUrl = assembleRequestUrl(REQUEST_URL, API_KEY, API_SECRET,"GET");
            Request request = new Request.Builder().url(authedRequestUrl).build();
            client.newWebSocket(request, this);

            InputStream is = this.inputStream;
            byte[] frame = new byte[122];
            int sleepInterval = 40;
            int status = WebsocketDemo.STATUS_BEGIN;
            int seq = 0;
            while (true) {
                if (isClose.get()){
                    System.out.println("session closed");
                    return;
                }
                int n = is.read(frame);
                if (n <= 0) {
                    status = WebsocketDemo.STATUS_END;
                }

				JsonObject req = new JsonObject();
				// generate header
				JsonObject header = new JsonObject();
				
				header.addProperty("did","");
				header.addProperty("imei","866402031869366");
				header.addProperty("net_isp","CMCC");
				header.addProperty("net_type","wifi");
				header.addProperty("status",status);
				header.addProperty("uid","");
				header.addProperty("app_id",appId);
				header.addProperty("imsi","");
				header.addProperty("mac","6c:92:bf:65:c6:14");
				header.addProperty("request_id","");
				
				// generater parameter
				JsonObject parameter = new JsonObject();
				
				JsonObject parameter_ist = new JsonObject();
				
				parameter_ist.addProperty("proc",0);
				parameter_ist.addProperty("dwa","wpgs");
				parameter_ist.addProperty("evl",0);
				parameter_ist.addProperty("nunum",0);
				JsonObject accept = new JsonObject();
				
				accept.addProperty("compress","raw");
				accept.addProperty("encoding","utf8");
				accept.addProperty("format","plain");
				parameter_ist.add("result",accept);
				parameter.add("ist",parameter_ist);
				
				// generate payload
				JsonObject payload = new JsonObject();
				
				JsonObject input = new JsonObject();
				
				input.addProperty("sample_rate",16000);
				input.addProperty("seq",seq++);
				input.addProperty("status",status);
				input.addProperty("audio",Base64.getEncoder().encodeToString(Arrays.copyOf(frame, n > 0 ? n : 0)));
				input.addProperty("bit_depth",16);
				input.addProperty("channels",1);
				input.addProperty("encoding","opus-wb");
				input.addProperty("frame_size",0);
				payload.add("input", input);
			
                req.add("header", header);
                // parameter should be send only in first frame
                if (status == WebsocketDemo.STATUS_BEGIN) {
                    req.add("parameter", parameter);
                }
				req.add("payload",payload);

                String paramStr = req.toString();
                this.inputDataQueue.put(paramStr);
                if (status == WebsocketDemo.STATUS_END) {
                    return;
                }
                if (status == WebsocketDemo.STATUS_BEGIN) {
                    status = WebsocketDemo.STATUS_CONTINUE;
                }
                Thread.sleep(sleepInterval);
            }
        } catch (Exception e) {
            e.printStackTrace();
            System.exit(1);
        }
    }

    @Override
    public void onFailure(WebSocket webSocket, Throwable t, Response response) {
        super.onFailure(webSocket, t, response);
        String bs = "";
        if (response != null && response.body() != null) {
            bs = response.toString();
        }
        t.printStackTrace();
        System.out.println(getDate() + ",onFailure:" + bs);
        this.close();
    }

    @Override
    public void onOpen(WebSocket webSocket, Response response) {
        super.onOpen(webSocket, response);
        this.webSocket = webSocket;
        try {
            System.out.println(getDate() + ",onOpen:" + response.body().string());
            new Thread(() -> {
                try {
                    start();
                } catch (InterruptedException e) {
                    e.printStackTrace();
                }
            }).start();

        } catch (IOException e) {
            e.printStackTrace();
        }
    }

    @Override
    public void onClosed(WebSocket webSocket, int code, String reason) {
        super.onClosed(webSocket, code, reason);
        System.out.println(String.format("%s,onClose=>code=%d,reason=%s", getDate(), code, reason));
        this.close();
    }

    @Override
    public void onMessage(WebSocket webSocket, String text) {
        super.onMessage(webSocket, text);
        System.out.println(getDate() + ",onMessage:" + text);

        Gson gson = new Gson();
        JsonObject resp = gson.fromJson(text, JsonObject.class);
        JsonObject header = resp.getAsJsonObject("header");
        if (header != null){
           if (header.get("code").getAsInt() != 0){
               this.close();
               System.out.println("error=========");
           }
        }

    }

    /**
     * 开始发送数据
     *
     * @throws InterruptedException 异常
     */
    private void start() throws InterruptedException {
        while (true) {
            String frame = this.inputDataQueue.take();
            if (frame.equals(STATUS_STOP + "")) {
                return;
            }
            try {
                if (this.isClose.get()) {
                    throw new Exception("send message on closed websocket");
                }
                if (this.webSocket != null) {
                    this.webSocket.send(frame);
                }
            } catch (Exception e) {
                e.printStackTrace();
                return;
            }
            System.out.println(String.format("%s,start send message=>%s", getDate(), frame));
        }
    }

    /**
     * 关闭流服务
     */
    public void close() {
        try {
            this.inputDataQueue.put(WebsocketDemo.STATUS_STOP + "");
        } catch (InterruptedException e) {
            e.printStackTrace();
        }
        if (this.webSocket != null) {
            this.webSocket.close(1000, "close");
        }
        this.isClose.set(true);
    }

    /**
     * 签名
     *
     * @param requestUrl url
     * @param apiKey     apiKey
     * @param apiSecret  apiSecret
     * @return url
     */
    public static String assembleRequestUrl(String requestUrl, String apiKey, String apiSecret,String method) {
        URL url = null;
        // 替换调schema前缀 ，原因是URL库不支持解析包含ws,wss schema的url
        String httpRequestUrl = requestUrl.replace("ws://", "http://").replace("wss://", "https://");
        try {
            url = new URL(httpRequestUrl);
            //获取当前日期并格式化
            SimpleDateFormat format = new SimpleDateFormat("EEE, dd MMM yyyy HH:mm:ss z", Locale.US);
            format.setTimeZone(TimeZone.getTimeZone("GMT"));
            String date = format.format(new Date());

            String host = url.getHost();
            if (url.getPort()  != -1) {
                host = host + ":" + String.valueOf(url.getPort());
            }
            StringBuilder builder = new StringBuilder("host: ").append(host).append("\n").//
                    append("date: ").append(date).append("\n").//
                    append(method).append(" ").
                    append(url.getPath()).append(" HTTP/1.1");
            Charset charset = Charset.forName("UTF-8");
            Mac mac = Mac.getInstance("hmacsha256");
            SecretKeySpec spec = new SecretKeySpec(apiSecret.getBytes(charset), "hmacsha256");
            mac.init(spec);
            byte[] hexDigits = mac.doFinal(builder.toString().getBytes(charset));
            String sha = Base64.getEncoder().encodeToString(hexDigits);

            String authorization = String.format("hmac username=\"%s\", algorithm=\"%s\", headers=\"%s\", signature=\"%s\"", apiKey, "hmac-sha256", "host date request-line", sha);
            String authBase = Base64.getEncoder().encodeToString(authorization.getBytes(charset));
            return String.format("%s?authorization=%s&host=%s&date=%s", requestUrl, URLEncoder.encode(authBase), URLEncoder.encode(host), URLEncoder.encode(date));

        } catch (Exception e) {
            throw new RuntimeException("assemble requestUrl error:" + e.getMessage());
        }
    }

    private SimpleDateFormat format = new SimpleDateFormat();

    public String getDate() {
        return format.format(new Date());
    }
}


