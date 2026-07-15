from http import HTTPStatus
from typing import Any
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.add_subscription_addon_body import AddSubscriptionAddonBody
from ...models.error import Error
from ...models.subscription_addon import SubscriptionAddon
from ...types import Response


def _get_kwargs(
    id: UUID,
    *,
    body: AddSubscriptionAddonBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/subscriptions/{id}/addons".format(
            id=quote(str(id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | SubscriptionAddon | None:
    if response.status_code == 201:
        response_201 = SubscriptionAddon.from_dict(response.json())

        return response_201

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())

        return response_404

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | SubscriptionAddon]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: AddSubscriptionAddonBody,
) -> Response[Error | SubscriptionAddon]:
    """Attach an add-on plan to a subscription

     Attaches an existing plan to the subscription as a priced add-on with a quantity. The subscription's
    base plan is unchanged; the add-on is billed as an extra line (add-on plan price × quantity, taxed
    independently) starting from the NEXT recurring invoice. The add-on plan's currency must match the
    subscription's base-plan currency. Mid-cycle proration is not applied in v1.

    Args:
        id (UUID):
        body (AddSubscriptionAddonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SubscriptionAddon]
    """

    kwargs = _get_kwargs(
        id=id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: AddSubscriptionAddonBody,
) -> Error | SubscriptionAddon | None:
    """Attach an add-on plan to a subscription

     Attaches an existing plan to the subscription as a priced add-on with a quantity. The subscription's
    base plan is unchanged; the add-on is billed as an extra line (add-on plan price × quantity, taxed
    independently) starting from the NEXT recurring invoice. The add-on plan's currency must match the
    subscription's base-plan currency. Mid-cycle proration is not applied in v1.

    Args:
        id (UUID):
        body (AddSubscriptionAddonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SubscriptionAddon
    """

    return sync_detailed(
        id=id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: AddSubscriptionAddonBody,
) -> Response[Error | SubscriptionAddon]:
    """Attach an add-on plan to a subscription

     Attaches an existing plan to the subscription as a priced add-on with a quantity. The subscription's
    base plan is unchanged; the add-on is billed as an extra line (add-on plan price × quantity, taxed
    independently) starting from the NEXT recurring invoice. The add-on plan's currency must match the
    subscription's base-plan currency. Mid-cycle proration is not applied in v1.

    Args:
        id (UUID):
        body (AddSubscriptionAddonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | SubscriptionAddon]
    """

    kwargs = _get_kwargs(
        id=id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: AddSubscriptionAddonBody,
) -> Error | SubscriptionAddon | None:
    """Attach an add-on plan to a subscription

     Attaches an existing plan to the subscription as a priced add-on with a quantity. The subscription's
    base plan is unchanged; the add-on is billed as an extra line (add-on plan price × quantity, taxed
    independently) starting from the NEXT recurring invoice. The add-on plan's currency must match the
    subscription's base-plan currency. Mid-cycle proration is not applied in v1.

    Args:
        id (UUID):
        body (AddSubscriptionAddonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | SubscriptionAddon
    """

    return (
        await asyncio_detailed(
            id=id,
            client=client,
            body=body,
        )
    ).parsed
