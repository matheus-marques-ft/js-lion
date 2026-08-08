import type { Emitter } from 'mitt';

import mitt from 'mitt';

import type { LunaMessage, LunaMessageEvents } from '@/types/postmessage.type';

import { LUNA_MESSAGE_TYPE } from '@/types/postmessage.type';

// Get all event types
export type LunaEventType = keyof LunaMessageEvents;

// Create event-to-data mapping type
type EventPayloadMap = {
  [K in LunaEventType]: LunaMessageEvents[K]['data'] extends undefined
    ? void
    : LunaMessageEvents[K]['data'];
};

const allEventTypes = Object.keys(LUNA_MESSAGE_TYPE) as LunaEventType[];

class LunaCommunicator<T extends EventPayloadMap = EventPayloadMap> {
  private mitt: Emitter<T>;
  private lunaId: string = '';
  private targetOrigin: string = '*';
  private protocol: string = '';

  constructor() {
    this.mitt = mitt<T>();
    this.setupMessageListener();
  }

  private setupMessageListener() {
    window.addEventListener('message', (event: MessageEvent) => {
      const message: LunaMessage = event.data;
      switch (message.name) {
        case LUNA_MESSAGE_TYPE.PING:
          this.lunaId = message.id;
          this.targetOrigin = event.origin;
          this.protocol = message.protocol;
          this.sendLuna(LUNA_MESSAGE_TYPE.PONG, '');
          console.log(
            `LunaCommunicator initialized with ID: ${this.lunaId}, Origin: ${this.targetOrigin}, Protocol: ${this.protocol}`,
          );
          break;
        default:
          // Handle other message types
          if (allEventTypes.includes(message.name as LunaEventType)) {
            const eventType = message.name as keyof T;
            const data = message as T[keyof T];
            this.mitt.emit(eventType, data);
          } else {
            console.warn(`Unhandled message type: ${message.name}`, message);
          }
      }
    });
  }

  // Send message to the target window
  public sendLuna<K extends keyof T>(name: K, data: T[K]) {
    if (!this.lunaId || !this.targetOrigin) {
      console.warn('Target window not set');
    }

    window.parent.postMessage({ name, id: this.lunaId, data }, this.targetOrigin);
  }

  // Listen for events
  public onLuna<K extends keyof T>(type: K, handler: (data: T[K]) => void) {
    this.mitt.on(type, handler);
  }

  // Remove a listener
  public offLuna<K extends keyof T>(type: K, handler?: (data: T[K]) => void) {
    this.mitt.off(type, handler);
  }

  // Listen for a one-time event
  public once<K extends keyof T>(type: K, handler: (data: T[K]) => void) {
    const onceHandler = (data: T[K]) => {
      handler(data);
      this.offLuna(type, onceHandler);
    };
    this.onLuna(type, onceHandler);
  }

  // Destroy the instance
  public destroy() {
    this.mitt.all.clear();
  }

  // Get all event types
  public getEventTypes(): Array<keyof T> {
    return Object.keys(this.mitt.all) as Array<keyof T>;
  }
}

export const lunaCommunicator = new LunaCommunicator<EventPayloadMap>();
