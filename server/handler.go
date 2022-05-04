package server

type Handler func(s *WsSession)

type HandlerChain []Handler
