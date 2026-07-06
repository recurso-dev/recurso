from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.handle_razorpay_webhook_body import HandleRazorpayWebhookBody
from ...models.handle_razorpay_webhook_response_200 import HandleRazorpayWebhookResponse200
from ...types import Response


def _get_kwargs(
    *,
    body: HandleRazorpayWebhookBody,
    x_razorpay_signature: str,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["X-Razorpay-Signature"] = x_razorpay_signature

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/webhooks/razorpay",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | HandleRazorpayWebhookResponse200 | None:
    if response.status_code == 200:
        response_200 = HandleRazorpayWebhookResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | HandleRazorpayWebhookResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: HandleRazorpayWebhookBody,
    x_razorpay_signature: str,
) -> Response[Error | HandleRazorpayWebhookResponse200]:
    """Razorpay webhook receiver

     Inbound receiver for Razorpay events (payment captured/failed, mandate lifecycle,
    virtual-account credits). The raw body must be signed with the
    `X-Razorpay-Signature` HMAC header; unsigned or tampered requests are rejected.
    This endpoint is called by Razorpay, not by API consumers.

    Args:
        x_razorpay_signature (str):
        body (HandleRazorpayWebhookBody): Raw Razorpay event payload.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | HandleRazorpayWebhookResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
        x_razorpay_signature=x_razorpay_signature,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: HandleRazorpayWebhookBody,
    x_razorpay_signature: str,
) -> Error | HandleRazorpayWebhookResponse200 | None:
    """Razorpay webhook receiver

     Inbound receiver for Razorpay events (payment captured/failed, mandate lifecycle,
    virtual-account credits). The raw body must be signed with the
    `X-Razorpay-Signature` HMAC header; unsigned or tampered requests are rejected.
    This endpoint is called by Razorpay, not by API consumers.

    Args:
        x_razorpay_signature (str):
        body (HandleRazorpayWebhookBody): Raw Razorpay event payload.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | HandleRazorpayWebhookResponse200
    """

    return sync_detailed(
        client=client,
        body=body,
        x_razorpay_signature=x_razorpay_signature,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: HandleRazorpayWebhookBody,
    x_razorpay_signature: str,
) -> Response[Error | HandleRazorpayWebhookResponse200]:
    """Razorpay webhook receiver

     Inbound receiver for Razorpay events (payment captured/failed, mandate lifecycle,
    virtual-account credits). The raw body must be signed with the
    `X-Razorpay-Signature` HMAC header; unsigned or tampered requests are rejected.
    This endpoint is called by Razorpay, not by API consumers.

    Args:
        x_razorpay_signature (str):
        body (HandleRazorpayWebhookBody): Raw Razorpay event payload.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | HandleRazorpayWebhookResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
        x_razorpay_signature=x_razorpay_signature,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: HandleRazorpayWebhookBody,
    x_razorpay_signature: str,
) -> Error | HandleRazorpayWebhookResponse200 | None:
    """Razorpay webhook receiver

     Inbound receiver for Razorpay events (payment captured/failed, mandate lifecycle,
    virtual-account credits). The raw body must be signed with the
    `X-Razorpay-Signature` HMAC header; unsigned or tampered requests are rejected.
    This endpoint is called by Razorpay, not by API consumers.

    Args:
        x_razorpay_signature (str):
        body (HandleRazorpayWebhookBody): Raw Razorpay event payload.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | HandleRazorpayWebhookResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
            x_razorpay_signature=x_razorpay_signature,
        )
    ).parsed
