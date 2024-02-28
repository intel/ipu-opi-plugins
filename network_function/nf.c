#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netinet/ip.h>
#include <netinet/if_ether.h>
#include <netinet/tcp.h>
#include <netinet/udp.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <net/if.h>
#include <sys/ioctl.h>
#include <linux/if_packet.h>

void printPacket(char *packet, int length) {
    int i;
    for (i = 0; i < length; i++) {
        printf("%02x ", (unsigned char)packet[i]);
        if ((i + 1) % 16 == 0) {
            printf("\n");
        }
    }
    printf("\n");
}

int main() {
    int rawSocket;
    struct sockaddr_ll sa_in, sa_out;
    struct ifreq ifr_in, ifr_out;
    // Create raw socket
    rawSocket = socket(AF_PACKET, SOCK_RAW, htons(ETH_P_ALL));
    if (rawSocket == -1) {
        perror("Failed to create raw socket");
        exit(1);
    }
    // Set the interface to receive packets
    strncpy(ifr_in.ifr_name, "net1", IFNAMSIZ);
    if (ioctl(rawSocket, SIOCGIFINDEX, &ifr_in) == -1) {
        perror("Failed to get interface index tap 1");
        close(rawSocket);
        exit(1);
    }
    // Set the interface to send packets
    strncpy(ifr_out.ifr_name, "net2", IFNAMSIZ);
    if (ioctl(rawSocket, SIOCGIFINDEX, &ifr_out) == -1) {
        perror("Failed to get interface index");
        close(rawSocket);
        exit(1);
    }
    // Bind socket to interface to receive packets
    memset(&sa_in, 0, sizeof(struct sockaddr_ll));
    sa_in.sll_family = AF_PACKET;
    sa_in.sll_protocol = htons(ETH_P_ALL);
    sa_in.sll_ifindex = ifr_in.ifr_ifindex;
    if (bind(rawSocket, (struct sockaddr *)&sa_in, sizeof(struct sockaddr_ll)) == -1) {
        perror("Failed to bind raw socket to interface");
        close(rawSocket);
        exit(1);
    }
    // Set socket address to send packets
    memset(&sa_out, 0, sizeof(struct sockaddr_ll));
    sa_out.sll_family = AF_PACKET;
    sa_out.sll_protocol = htons(ETH_P_ALL);
    sa_out.sll_ifindex = ifr_out.ifr_ifindex;
    static int i = 0;

    // Receive and send packets
    while (1) {
        char buffer[65536];
        ssize_t length;

        // Receive packet
        length = recvfrom(rawSocket, buffer, sizeof(buffer), 0, NULL, NULL);
        if (length == -1) {
            perror("Failed to receive packet");
            close(rawSocket);
            exit(1);

	    i++;
        printf("received frame. Sent frame count %d\n", i);
        printf ("length = %d \n", length);
        printPacket(buffer, length);

        // Extract the Ethernet packet type
        unsigned short etherType = ntohs(*(unsigned short *)(buffer + 12));
        // Print the Ethernet packet type
        printf("Ethernet Packet Type: 0x%04x\n", etherType);        

        // Extract the source MAC address
        unsigned char *sourceMac = buffer + 6;
        // Print the source MAC address
        printf("Source MAC Address: %02x:%02x:%02x:%02x:%02x:%02x\n",
           sourceMac[0], sourceMac[1], sourceMac[2], sourceMac[3], sourceMac[4], sourceMac[5]);

        }   

      // Send packet
        if (sendto(rawSocket, buffer, length, 0, (struct sockaddr *)&sa_out, sizeof(struct sockaddr_ll)) == -1) {
            perror("Failed to send packet");
            close(rawSocket);
            exit(1);
        } 
    }
    // Close socket
    close(rawSocket);
    return 0;
}